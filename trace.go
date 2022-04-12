package fgtrace

import (
	"bytes"
	"errors"
	"io"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/DataDog/gostackparse"
)

// DefaultHz is the default number of goroutine profiles captured per second
// when no other value is given via WithHz.
const DefaultHz = 99

// TraceFile starts capturing a goroutine trace to the given filename and
// returns a stop function that finishes the recording. The stop function
// returns nil if the trace was written successfully. Calling stop more than
// once will return an error.
func TraceFile(filename string, opts ...TracerOption) func() error {
	file, err := os.Create(filename)
	if err != nil {
		return func() error { return err }
	}
	t := Trace(file, opts...)
	return func() (err error) {
		err = t.Stop()
		if closeErr := file.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
		return
	}
}

// Trace starts capturing a goroutine profile trace with the given opts to dst.
// Calling Stop() on the returned tracer finishes the recording.
func Trace(dst io.Writer, opts ...TracerOption) *Tracer {
	t := &Tracer{
		hz:      DefaultHz,
		dst:     dst,
		stop:    make(chan struct{}),
		stopped: make(chan error, 1),
	}
	for _, opt := range opts {
		opt(t)
	}
	t.start()
	return t
}

type TracerOption func(*Tracer)

// WithHz sets how many goroutine profiles are captured per second. The default
// is DefaultHz.
func WithHz(hz int) TracerOption {
	return func(t *Tracer) { t.hz = hz }
}

// Tracer is used to capture a goroutine profile traces.
type Tracer struct {
	hz  int       // sampling frequency
	dst io.Writer // trace destination

	err     error         // the error that caused the tracer to stop
	stop    chan struct{} // closed to initiate stop
	stopped chan error    // messaged to confirm stop completed
	enc     *encoder      // trace event format encoder
	samples int           // used for internal testing only right now
}

// start begins writing to dst and launches the background trace goroutine.
func (t *Tracer) start() {
	if t.enc, t.err = newEncoder(t.dst); t.err != nil {
		return
	} else if t.err = t.enc.CustomMeta("hz", t.hz); t.err != nil {
		return
	}

	go func() { t.stopped <- t.trace() }()
}

// Stop stops the tracer and returns nil on success. An error indicates a
// problem with writing to dst. Calling Stop() more than once returns the
// previous error or an error indicating that the tracer has already been
// stopped.
func (t *Tracer) Stop() error {
	if t.err != nil {
		return t.err
	}

	close(t.stop)
	err := <-t.stopped
	// TODO(fg) does the trace format support writing error messages? if yes,
	// we should probably attempt to write the error to the file as well.
	if finishErr := t.enc.Finish(); finishErr != nil && t.err == nil {
		err = finishErr
	}

	if err != nil {
		t.err = err
	} else {
		// To be returned if Stop() is called more than once.
		t.err = errors.New("tracer is already stopped")
	}

	return err
}

// trace is the background goroutine that takes goroutine profiles and converts
// them to trace events.
func (t *Tracer) trace() error {
	var (
		tick           = time.NewTicker(time.Second / time.Duration(t.hz))
		start          = time.Now()
		now            = start
		prevGoroutines = make(map[int]*gostackparse.Goroutine)
		prof           goroutineProfiler
	)
	defer tick.Stop()

	for {
		t.samples++
		ts := now.Sub(start).Seconds() * 1e6
		goroutines, err := prof.Goroutines()
		if err != nil {
			return err
		}
		currentGoroutines := make(map[int]*gostackparse.Goroutine, len(prevGoroutines))
		for _, current := range goroutines {
			currentGoroutines[current.ID] = current
			prev := prevGoroutines[current.ID]
			if err := t.enc.Encode(ts, prev, current); err != nil {
				return err
			}
		}
		for _, prev := range prevGoroutines {
			if _, ok := currentGoroutines[prev.ID]; ok {
				continue
			}
			if err := t.enc.Encode(ts, prev, nil); err != nil {
				return err
			}
		}
		prevGoroutines = currentGoroutines

		// Sleep until next tick comes up or the tracer is stopped.
		select {
		case now = <-tick.C:
		case <-t.stop:
			ts := time.Since(start).Seconds() * 1e6
			for _, prev := range prevGoroutines {
				if err := t.enc.Encode(ts, prev, nil); err != nil {
					return err
				}
			}
			return nil
		}
	}
}

type goroutineProfiler struct {
	buf []byte
}

func (g *goroutineProfiler) Goroutines() ([]*gostackparse.Goroutine, error) {
	if g.buf == nil {
		g.buf = make([]byte, 16*1024)
	}
	for {
		n := runtime.Stack(g.buf, true)
		if n < len(g.buf) {
			gs, errs := gostackparse.Parse(bytes.NewReader(g.buf[:n]))
			if len(errs) > 0 {
				return gs, errs[0]
			}
			addVirualStateFrames(gs)
			gs = excludeSelf(gs)
			return gs, nil
		}
		g.buf = make([]byte, 2*len(g.buf))
	}
}

// pkgName is the full name of the current pkg.
var pkgName = (func() string {
	pc, _, _, _ := runtime.Caller(0)
	frames := runtime.CallersFrames([]uintptr{pc})
	frame, _ := frames.Next()
	fn := frame.Func.Name()
	parts := strings.Split(fn, "/")
	parts[len(parts)-1], _, _ = strings.Cut(parts[len(parts)-1], ".")
	return strings.Join(parts, "/")
})()

func excludeSelf(gs []*gostackparse.Goroutine) []*gostackparse.Goroutine {
	newGS := make([]*gostackparse.Goroutine, 0, len(gs))
	for _, g := range gs {
		include := true
		for _, f := range g.Stack {
			if strings.HasPrefix(f.Func, pkgName) {
				include = false
				break
			}
		}
		if include {
			newGS = append(newGS, g)
		}
	}
	return newGS
}

func addVirualStateFrames(gs []*gostackparse.Goroutine) {
	for _, g := range gs {
		state := g.State
		if state == "runnable" {
			// Taking a goroutine profile puts all running goroutines into runnable
			// state. So let's indicate that we can't be sure of their real state,
			// but that it's most likely running instead of runnable.
			state = "running/runnable"
		}
		g.Stack = append(g.Stack, &gostackparse.Frame{
			Func: state,
			File: "runtime",
			Line: 1,
		})
	}
}
