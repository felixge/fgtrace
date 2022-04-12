package fgtrace

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/DataDog/gostackparse"
)

func newEncoder(w io.Writer) (*encoder, error) {
	e := &encoder{
		w:     w,
		json:  *json.NewEncoder(w),
		first: true,
	}
	_, err := w.Write([]byte("["))
	return e, err
}

type event struct {
	Name string `json:"name,omitempty"`
	Ph   string `json:"ph,omitempty"`
	// Ts is the tracing clock timestamp of the event. The timestamps are
	// provided at microsecond granularity.
	Ts   float64                `json:"ts"`
	Pid  int64                  `json:"pid,omitempty"`
	Tid  int64                  `json:"tid,omitempty"`
	Args map[string]interface{} `json:"args,omitempty"`
}

type encoder struct {
	w     io.Writer
	json  json.Encoder
	first bool
}

func (e *encoder) CustomMeta(name string, value interface{}) error {
	ev := event{
		Name: name,
		Ph:   "M",
		Args: map[string]interface{}{name: value},
	}
	return e.encode(&ev)
}

func (e *encoder) Encode(ts float64, prev, current *gostackparse.Goroutine) error {
	ev := event{Ts: ts, Tid: 1}
	prevLen := 0
	if prev != nil {
		prevLen = len(prev.Stack)
		ev.Pid = int64(prev.ID)
	}
	currentLen := 0
	if current != nil {
		currentLen = len(current.Stack)
		ev.Pid = int64(current.ID)
	}

	if prev == nil {
		metaEv := ev
		metaEv.Ts = 0
		metaEv.Name = "process_name"
		metaEv.Ph = "M"
		name := fmt.Sprintf("G%d", current.ID)
		if current.CreatedBy != nil {
			name += " " + current.CreatedBy.Func
		}
		metaEv.Args = map[string]interface{}{"name": name}
		if err := e.encode(&metaEv); err != nil {
			return err
		}
	}

	// Determine the number of stack frames that are identical between prev and
	// current going from root frame (e.g. main) to the leaf frame.
	commonDepth := prevLen
	for i := 0; i < prevLen; i++ {
		ci := currentLen - i - 1
		pi := prevLen - i - 1
		if ci < 0 || prev.Stack[pi].Func != current.Stack[ci].Func {
			commonDepth = i
			break
		}
	}

	// Emit end events for prev stack frames that are no longer part of the
	// current stack going from leaf to root frame.
	for pi := 0; pi < prevLen-commonDepth; pi++ {
		ev.Ph = "E"
		ev.Name = prev.Stack[pi].Func
		if err := e.encode(&ev); err != nil {
			return err
		}
	}

	// Emit start events for current stack frames that were not part of the prev
	// stack going from root to leaf frame.
	for i := commonDepth; i < currentLen; i++ {
		ci := currentLen - i - 1
		ev.Ph = "B"
		ev.Name = current.Stack[ci].Func
		if err := e.encode(&ev); err != nil {
			return err
		}
	}
	return nil
}

func (e *encoder) encode(ev *event) error {
	if !e.first {
		if _, err := e.w.Write([]byte(",")); err != nil {
			return err
		}
	} else {
		e.first = false
	}
	return e.json.Encode(ev)
}

func (e *encoder) Finish() error {
	_, err := e.w.Write([]byte("]"))
	return err
}
