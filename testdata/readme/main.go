package main

import (
	"bytes"
	"crypto/sha1"
	"flag"
	"fmt"
	"io"
	"net/http"
	"runtime/trace"
	"time"

	"github.com/felixge/fgprof"
	"github.com/felixge/fgtrace"
)

type Flags struct {
	Runtime bool
	Fgprof  bool
	Fgtrace bool
}

func main() {
	var f Flags
	flag.BoolVar(&f.Runtime, "runtime", false, "Capture runtime trace")
	flag.BoolVar(&f.Fgprof, "fgprof", false, "Capture fgprof profile")
	flag.BoolVar(&f.Fgtrace, "fgtrace", false, "Capture fgtrace trace")
	flag.Parse()
	stop := startTraces(f)
	defer stop()

	start := time.Now()
	for i := 0; i < 10; i++ {
		time.Sleep(10 * time.Millisecond)
	}
	fmt.Printf("time.Since(start): %v\n", time.Since(start))

	res, err := http.Get("https://github.com/")
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, res.Body); err != nil {
		panic(err)
	}

	for i := 0; i < 1000; i++ {
		sha1.Sum(buf.Bytes())
	}
}

func startTraces(f Flags) func() {
	var startFns []func() error
	var stopFns []func() error
	if f.Runtime {
		startFns = append(startFns, func() error {
			return trace.Start(fgtrace.File("runtime-trace.json"))
		})
		stopFns = append(stopFns, func() error {
			trace.Stop()
			return nil
		})
	}
	if f.Fgprof {
		var stopFn func() error
		startFns = append(startFns, func() error {
			stopFn = fgprof.Start(fgtrace.File("fgprof.pprof"), fgprof.FormatPprof)
			return nil
		})
		stopFns = append(stopFns, func() error {
			return stopFn()
		})

	}
	if f.Fgtrace {
		var fgTrace *fgtrace.Trace
		startFns = append(startFns, func() error {
			fgTrace = fgtrace.Config{Hz: 1000}.Trace()
			return nil
		})
		stopFns = append(stopFns, func() error { return fgTrace.Stop() })
	}

	mustRun := func(fns []func() error) {
		for _, fn := range fns {
			if err := fn(); err != nil {
				panic(err)
			}
		}
	}

	mustRun(startFns)
	return func() { mustRun(stopFns) }
}
