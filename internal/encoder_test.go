package internal

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/DataDog/gostackparse"
	"github.com/stretchr/testify/require"
)

func Test_writeEvents(t *testing.T) {
	tests := []struct {
		Name    string
		Ts      int64
		Prev    *gostackparse.Goroutine
		Current *gostackparse.Goroutine
		Want    []Event
	}{
		{
			Name:    "initial",
			Ts:      1000,
			Current: newTestGoroutine(42, "foo", "main"),
			Want: []Event{
				{Name: "process_name", Ph: "M", Ts: 0, Pid: 42, Tid: 1, Args: map[string]interface{}{"name": "G42"}},
				{Name: "main", Ph: "B", Ts: 1000, Pid: 42, Tid: 1},
				{Name: "foo", Ph: "B", Ts: 1000, Pid: 42, Tid: 1},
			},
		},

		{
			Name:    "call",
			Ts:      2000,
			Prev:    newTestGoroutine(42, "foo", "main"),
			Current: newTestGoroutine(42, "baz", "bar", "foo", "main"),
			Want: []Event{
				{Name: "bar", Ph: "B", Ts: 2000, Pid: 42, Tid: 1},
				{Name: "baz", Ph: "B", Ts: 2000, Pid: 42, Tid: 1},
			},
		},

		{
			Name:    "return",
			Ts:      3000,
			Prev:    newTestGoroutine(42, "baz", "bar", "foo", "main"),
			Current: newTestGoroutine(42, "foo", "main"),
			Want: []Event{
				{Name: "baz", Ph: "E", Ts: 3000, Pid: 42, Tid: 1},
				{Name: "bar", Ph: "E", Ts: 3000, Pid: 42, Tid: 1},
			},
		},

		{
			Name:    "returnAndCall",
			Ts:      4000,
			Prev:    newTestGoroutine(42, "baz", "bar", "foo", "main"),
			Current: newTestGoroutine(42, "foobar", "foo", "main"),
			Want: []Event{
				{Name: "baz", Ph: "E", Ts: 4000, Pid: 42, Tid: 1},
				{Name: "bar", Ph: "E", Ts: 4000, Pid: 42, Tid: 1},
				{Name: "foobar", Ph: "B", Ts: 4000, Pid: 42, Tid: 1},
			},
		},

		{
			Name: "finish",
			Ts:   5000,
			Prev: newTestGoroutine(42, "foo", "main"),
			Want: []Event{
				{Name: "foo", Ph: "E", Ts: 5000, Pid: 42, Tid: 1},
				{Name: "main", Ph: "E", Ts: 5000, Pid: 42, Tid: 1},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			e, err := NewEncoder(buf)
			require.NoError(t, err)
			require.NoError(t, e.Encode(float64(test.Ts), test.Prev, test.Current))
			require.NoError(t, e.Finish())
			var got []Event
			require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
			require.Equal(t, test.Want, got)
		})
	}
}

func newTestGoroutine(gid int, stack ...string) *gostackparse.Goroutine {
	frames := make([]gostackparse.Frame, len(stack))
	g := &gostackparse.Goroutine{
		ID:    gid,
		Stack: make([]*gostackparse.Frame, len(stack)),
	}
	for i, fn := range stack {
		frames[i].Func = fn
		g.Stack[i] = &frames[i]
	}
	return g
}
