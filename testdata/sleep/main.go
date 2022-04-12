package main

import (
	"time"

	"github.com/felixge/fgtrace"
)

func main() {
	defer fgtrace.TraceFile("sleep.json")()

	go sleep100()
	sleep200()
}

func sleep100() {
	time.Sleep(100 * time.Millisecond)
}

func sleep200() {
	time.Sleep(200 * time.Millisecond)
}
