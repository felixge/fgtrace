package fgtrace

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func ExampleTraceFile() {
	stop := TraceFile("trace.json")
	defer stop()

	// Code to be traced
}

func ExampleTrace() {
	tracer := Trace(os.Stdout)
	defer tracer.Stop()

	// Code to be traced
}

func TestTracer(t *testing.T) {
	defer goleak.VerifyNone(t)

	t.Run("produces-json", func(t *testing.T) {
		var buf bytes.Buffer
		tracer := Trace(&buf)
		require.NoError(t, tracer.Stop())
		var val interface{}
		require.NoError(t, json.Unmarshal(buf.Bytes(), &val))
		require.IsType(t, []interface{}{}, val)
	})

	t.Run("stop-returns-error", func(t *testing.T) {
		ew := errWriter{errors.New("whups")}
		tracer := Trace(ew)
		require.Equal(t, ew.err, tracer.Stop())
	})

	t.Run("with-hz-default", func(t *testing.T) {
		tracer, stop := recordTrace()
		require.Equal(t, DefaultHz, tracer.hz)
		time.Sleep(time.Second / 10)
		tr, err := stop()
		require.NoError(t, err)
		require.InDelta(t, tracer.hz/10, tracer.samples, 3)
		require.Equal(t, DefaultHz, tr.MetaHz())
	})

	t.Run("with-hz-custom", func(t *testing.T) {
		const customHz = 1000
		tracer, stop := recordTrace(WithHz(customHz))
		require.Equal(t, customHz, tracer.hz)
		time.Sleep(time.Second / 10)
		tr, err := stop()
		require.NoError(t, err)
		require.InDelta(t, tracer.hz/10, tracer.samples, 3)
		require.Equal(t, customHz, tr.MetaHz())
	})

	t.Run("hide-self", func(t *testing.T) {
		_, stop := recordTrace()
		time.Sleep(time.Second / 10)
		record, err := stop()
		require.NoError(t, err)
		for _, e := range record.events {
			if strings.Contains(e.Name, pkgName) {
				t.Fatal(e.Name)
			}
		}
	})
}

func Test_pkgName(t *testing.T) {
	require.NotEmpty(t, pkgName)
	require.True(t, strings.HasSuffix(pkgName, "/fgtrace"))
	require.True(t, strings.HasPrefix(pkgName, "github.com/"))
}

type errWriter struct{ err error }

func (e errWriter) Write(p []byte) (int, error) { return 0, e.err }

func recordTrace(opts ...TracerOption) (*Tracer, func() (*traceRecord, error)) {
	var buf bytes.Buffer
	t := Trace(&buf, opts...)
	return t, func() (*traceRecord, error) {
		err := t.Stop()
		tr, parseErr := parseTrace(&buf)
		if parseErr != nil && err == nil {
			err = parseErr
		}
		return tr, err
	}
}

func parseTrace(r io.Reader) (*traceRecord, error) {
	var tr traceRecord
	return &tr, json.NewDecoder(r).Decode(&tr.events)
}

type traceRecord struct {
	events []event
}

func (t *traceRecord) MetaHz() int {
	for _, e := range t.events {
		if e.Ph == "M" && e.Name == "hz" {
			hz, _ := e.Args["hz"].(float64)
			return int(hz)
		}
	}
	return 0
}
