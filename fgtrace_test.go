package fgtrace

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/felixge/fgtrace/internal"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func ExampleConfig_ServeHTTP() {
	// Serve traces via /debug/fgtrace endpoint
	http.DefaultServeMux.Handle("/debug/fgtrace", Config{})
	http.ListenAndServe(":1234", nil)
}

func ExampleConfig_Trace() {
	// Write trace to the default fgtrace.json file
	defer Config{}.Trace().Stop()

	// Write high-resolution trace
	defer Config{Hz: 10000}.Trace().Stop()

	// Write trace to a custom file
	defer Config{Dst: File("path/to/myfile.json")}.Trace().Stop()

	// Write trace to a buffer
	var buf bytes.Buffer
	defer Config{Dst: Writer(&buf)}.Trace().Stop()
}

func TestConfig(t *testing.T) {
	defer goleak.VerifyNone(t)

	t.Run("WithDefaults", func(t *testing.T) {
		defaults := Config{}.WithDefaults()
		require.Equal(t, Config{
			Hz:           defaultHz,
			Dst:          File(defaultFile),
			HTTPDuration: defaultHTTPDuration,
			StateFrames:  defaultStateFrames,
			IncludeSelf:  false,
		}, defaults)

		noDefaults := Config{
			Dst:          os.Stdout,
			Hz:           23,
			HTTPDuration: 42 * time.Second,
			StateFrames:  StateFramesNo,
			IncludeSelf:  true,
		}
		require.Equal(t, noDefaults, noDefaults.WithDefaults())
	})

	t.Run("Trace", func(t *testing.T) {
		t.Run("produces-json", func(t *testing.T) {
			buf := &bytes.Buffer{}
			trace := Config{Dst: Writer(buf)}.Trace()
			require.NoError(t, trace.Stop())
			var val interface{}
			require.NoError(t, json.Unmarshal(buf.Bytes(), &val))
			require.IsType(t, []interface{}{}, val)
		})

		t.Run("stop-returns-error", func(t *testing.T) {
			ew := internal.ErrWriter{Err: errors.New("whups")}
			trace := Config{Dst: Writer(ew)}.Trace()
			require.Equal(t, ew.Err, trace.Stop())
		})

		t.Run("Hz", func(t *testing.T) {
			testHz := func(t *testing.T, hz, attempt int) bool {
				testDt := time.Duration(math.Pow(2, float64(attempt)) * float64(time.Second/10))
				buf := &bytes.Buffer{}
				conf := Config{
					Dst:         Writer(buf),
					Hz:          hz,
					IncludeSelf: true,
				}
				trace := conf.Trace()
				workloadSimulator(hz, testDt)
				require.NoError(t, trace.Stop())
				data, err := internal.Unmarshal(buf.Bytes())
				require.NoError(t, err)
				require.Equal(t, conf.Hz, data.MetaHz())
				callCount := data.Filter(func(e *internal.Event) bool {
					return e.Ph == "B" &&
						(strings.HasSuffix(e.Name, "fgtrace.workloadA") ||
							strings.HasSuffix(e.Name, "fgtrace.workloadB"))
				}).Len()
				wantCount := hz / int(time.Second/testDt)
				wantEpsilon := 0.2
				if attempt < 7 {
					epsilon := math.Abs(float64(wantCount-callCount)) / math.Abs(float64(wantCount))
					return epsilon <= wantEpsilon
				} else {
					require.InEpsilon(t, wantCount, callCount, wantEpsilon)
					return true
				}
			}

			for _, hz := range []int{10, 100, 200, 1000, 10000} {
				t.Run(fmt.Sprintf("%d", hz), func(t *testing.T) {
					for attempt := 0; ; attempt++ {
						if testHz(t, hz, attempt) {
							return
						}
					}
				})
			}
		})

		t.Run("IncludeSelf", func(t *testing.T) {
			test := func(t *testing.T, includeSelf bool) int {
				buf := &bytes.Buffer{}
				conf := Config{
					Dst:         Writer(buf),
					IncludeSelf: includeSelf,
				}.WithDefaults()
				trace := conf.Trace()
				workloadSimulator(conf.Hz, time.Second/10)
				require.NoError(t, trace.Stop())
				data, err := internal.Unmarshal(buf.Bytes())
				require.NoError(t, err)
				return data.Filter(func(e *internal.Event) bool {
					return strings.Contains(e.Name, internal.ModulePath())
				}).Len()
			}

			t.Run("true", func(t *testing.T) {
				callCount := test(t, true)
				require.GreaterOrEqual(t, callCount, 1)
			})

			t.Run("false", func(t *testing.T) {
				callCount := test(t, false)
				require.Equal(t, 0, callCount)
			})
		})

		t.Run("StateFrames", func(t *testing.T) {
			test := func(t *testing.T, f StateFrames) *internal.Node {
				buf := &bytes.Buffer{}
				conf := Config{
					Dst:         Writer(buf),
					IncludeSelf: true,
					StateFrames: f,
				}.WithDefaults()
				trace := conf.Trace()
				workloadSimulator(conf.Hz, time.Second/10)
				require.NoError(t, trace.Stop())
				data, err := internal.Unmarshal(buf.Bytes())
				require.NoError(t, err)
				return data.CallGraph()
			}

			t.Run("StateFramesRoot", func(t *testing.T) {
				graph := test(t, StateFramesRoot)
				require.Equal(t, graph.Children[0].Func, "running")
				require.False(t, graph.HasLeaf("running"))
			})

			t.Run("StateFramesLeaf", func(t *testing.T) {
				graph := test(t, StateFramesLeaf)
				require.NotEqual(t, graph.Children[0].Func, "running")
				require.True(t, graph.HasLeaf("running"))
			})

			t.Run("StateFramesNo", func(t *testing.T) {
				graph := test(t, StateFramesNo)
				require.NotEqual(t, graph.Children[0].Func, "running")
				require.False(t, graph.HasLeaf("running"))
			})
		})
	})
}

// workloadSimulator calls workloadA followed by workloadB in a loop. Each call
// takes 1/sampleHz to complete, so the effective frequency of the whole loop
// is sampleHz/2 which is the [nyquist frequency] of sampleHz, i.e. the highest
// frequency that a profiler operating at sampleHz can reliably measure.
//
// [nyquist frequency]: https://www.techtarget.com/whatis/definition/Nyquist-Theorem
func workloadSimulator(sampleHz int, dt time.Duration) {
	callDuration := time.Second / time.Duration(sampleHz)
	stopCh := make(chan struct{})
	time.AfterFunc(dt, func() { close(stopCh) })
	for i := 1; ; i++ {
		workloadA(callDuration, stopCh)
		workloadB(callDuration, stopCh)
		select {
		case <-stopCh:
			return
		default:
		}
	}
}

func workloadA(dt time.Duration, stopCh chan struct{}) {
	sleep(dt, stopCh)
}

func workloadB(dt time.Duration, stopCh chan struct{}) {
	sleep(dt, stopCh)
}

func sleep(dt time.Duration, stopCh chan struct{}) {
	select {
	case <-time.After(dt):
	case <-stopCh:
	}
}
