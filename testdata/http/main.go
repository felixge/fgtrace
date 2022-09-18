package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/felixge/fgtrace"
)

func main() {
	defer fgtrace.Config{Dst: fgtrace.File("fgtrace.json"), Hz: 10000}.Trace().Stop()

	start := time.Now()
	request1()
	fmt.Fprintf(os.Stderr, "req1: %s\n", time.Since(start))

	start = time.Now()
	request2()
	fmt.Fprintf(os.Stderr, "req2: %s\n", time.Since(start))
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
