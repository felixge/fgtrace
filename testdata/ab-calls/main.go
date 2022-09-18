package main

import (
	"time"

	"github.com/felixge/fgtrace"
)

func main() {
	defer fgtrace.Config{Hz: 1000}.Trace().Stop()
	time.Sleep(2 * time.Millisecond)
	workloadSimulator(200, time.Second)
}

// workloadSimulator calls workloadA followed by workloadB for 1/Hz each in a
// loop and returns after dt has passed.
func workloadSimulator(hz int, dt time.Duration) {
	callDuration := time.Second / time.Duration(hz)
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
