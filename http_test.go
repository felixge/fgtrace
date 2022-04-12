package fgtrace

import (
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func TestHandler(t *testing.T) {
	defer goleak.VerifyNone(t)

	t.Run("duration", func(t *testing.T) {
		for _, duration := range []time.Duration{
			100 * time.Millisecond,
			200 * time.Millisecond,
		} {
			rr := httptest.NewRecorder()
			r := httptest.NewRequest("GET", fmt.Sprintf("/?seconds=%f", duration.Seconds()), nil)
			start := time.Now()
			Handler().ServeHTTP(rr, r)
			dt := time.Since(start)
			require.InDelta(t, duration, dt, float64(25*time.Millisecond))
		}
	})

	t.Run("hz", func(t *testing.T) {
		hz := 123
		rr := httptest.NewRecorder()
		r := httptest.NewRequest("GET", fmt.Sprintf("/?hz=%d", hz), nil)
		Handler().ServeHTTP(rr, r)
		tr, err := parseTrace(rr.Body)
		require.NoError(t, err)
		require.Equal(t, hz, tr.MetaHz())
	})
}
