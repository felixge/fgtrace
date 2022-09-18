package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/felixge/fgtrace"
)

func main() {
	defer fgtrace.Config{Dst: fgtrace.File("fgtrace.json"), Hz: 10000}.Trace().Stop()

	stop := make(chan struct{})
	go jsonHog(stop)
	time.Sleep(1000 * time.Millisecond)
	stop <- struct{}{}
}

func jsonHog(stop chan struct{}) {
	for i := 0; ; i++ {
		var m interface{}
		json.Unmarshal([]byte(`{"foo": [1,true,3]}`), &m)
		select {
		case <-stop:
			fmt.Fprintf(os.Stderr, "i: %v\n", i)
			return
		default:
		}
	}
}
