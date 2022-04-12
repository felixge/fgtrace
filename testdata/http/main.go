package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/felixge/fgtrace"
)

func main() {
	//if err := trace.Start(os.Stdout); err != nil {
	//panic(err)
	//}
	//defer trace.Stop()
	stop := fgtrace.TraceFile("http.json", fgtrace.WithHz(1000))
	defer stop()

	go waitChan()

	start := time.Now()
	request1()
	fmt.Fprintf(os.Stderr, "req1: %s\n", time.Since(start))
	start = time.Now()
	request2()
	fmt.Fprintf(os.Stderr, "req2: %s\n", time.Since(start))
}

func waitChan() {
	ch := make(chan struct{})
	<-ch
}

func Stack() []byte {
	buf := make([]byte, 1024)
	for {
		n := runtime.Stack(buf, true)
		if n < len(buf) {
			return buf[:n]
		}
		buf = make([]byte, 2*len(buf))
	}
}

func request1() {
	res, err := http.Get("https://example.org/")
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()
	io.Copy(ioutil.Discard, res.Body)
	fmt.Fprintf(os.Stderr, "res.StatusCode: %v\n", res.StatusCode)
}

func request2() {
	res, err := http.Get("https://example.org/")
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()
	io.Copy(ioutil.Discard, res.Body)
	fmt.Fprintf(os.Stderr, "res.StatusCode: %v\n", res.StatusCode)
}
