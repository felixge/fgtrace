package fgtrace

import (
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// DefaultDuration is the default duration for goroutine profile tracing used
// by the Handler() function.
const DefaultDuration = 30 * time.Second

// Handler returns an http handler that captures a goroutine profile trace for
// DefaultDuration at DefaultHz and sends it as a reply. The defaults can be
// overwritten using the "seconds" and "hz" query parameters.
func Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var (
			seconds float64
			hz      int
			err     error
		)

		// parse seconds param
		if v := r.URL.Query().Get("seconds"); v == "" {
			seconds = float64(DefaultDuration)
		} else if seconds, err = strconv.ParseFloat(v, 64); err != nil || seconds <= 0 {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "bad seconds: %q: %s\n", v, err)
			return
		}

		// parse hz param
		if v := r.URL.Query().Get("hz"); v == "" {
			hz = DefaultHz
		} else if hz, err = strconv.Atoi(v); err != nil || hz <= 0 {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "bad hz: %q: %s\n", v, err)
			return
		}

		t := Trace(w, WithHz(hz))
		defer t.Stop()
		time.Sleep(time.Duration(seconds * float64(time.Second)))
	})
}
