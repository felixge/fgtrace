package fgtrace

// type Handler struct {
// 	// Duration is the default duration for traces served via http.
// 	// WithDefaults() sets it to 30s if it is 0.
// 	Duration time.Duration
// 	// Hz determines how often the stack traces of all goroutines are captured
// 	// per second. WithDefaults() sets it to 99 Hz if it is 0.
// 	Hz int
// }

// func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

// }

// type HttpConfig struct {
// 	// Duration is the default duration for traces served via http.
// 	// WithDefaults() sets it to 30s if it is 0.
// 	Duration time.Duration
// 	// Hz determines how often the stack traces of all goroutines are captured
// 	// per second. WithDefaults() sets it to 99 Hz if it is 0.
// 	Hz int
// }

// // WithDefaults returns a copy of c with default values applied as described
// // in the type documentation.
// func (c HttpConfig) WithDefaults() HttpConfig {
// 	if c.Duration == 0 {
// 		c.Duration = 30 * time.Second
// 	}
// 	if c.Hz == 0 {
// 		c.Hz = defaultHz
// 	}
// 	return c
// }

// DefaultDuration is the default duration for goroutine profile tracing used
// by the Handler() function.
// const DefaultDuration = 30 * time.Second

// Handler returns an http handler that captures a goroutine profile trace for
// DefaultDuration at DefaultHz and sends it as a reply. The defaults can be
// overwritten using the "seconds" and "hz" query parameters.
// func HandlerFunc() http.Handler {
// 	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		// var (
// 		// 	seconds float64
// 		// 	hz      int
// 		// 	err     error
// 		// )

// 		// // parse seconds param
// 		// if v := r.URL.Query().Get("seconds"); v == "" {
// 		// 	seconds = DefaultDuration.Seconds()
// 		// } else if seconds, err = strconv.ParseFloat(v, 64); err != nil || seconds <= 0 {
// 		// 	w.WriteHeader(http.StatusBadRequest)
// 		// 	fmt.Fprintf(w, "bad seconds: %q: %s\n", v, err)
// 		// 	return
// 		// }

// 		// // parse hz param
// 		// if v := r.URL.Query().Get("hz"); v == "" {
// 		// 	hz = defaultHz
// 		// } else if hz, err = strconv.Atoi(v); err != nil || hz <= 0 {
// 		// 	w.WriteHeader(http.StatusBadRequest)
// 		// 	fmt.Fprintf(w, "bad hz: %q: %s\n", v, err)
// 		// 	return
// 		// }

// 		// t := Trace(w, WithHz(hz))
// 		// defer t.Stop()
// 		// time.Sleep(time.Duration(seconds * float64(time.Second)))
// 	})
// }
