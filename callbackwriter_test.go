package golog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestCallbackWriter_WriteError_nil(t *testing.T) {
	var gotAttribs Attribs
	config := NewCallbackWriterConfig(func(timestamp time.Time, level Level, prefix, text string, attribs Attribs) {
		gotAttribs = attribs.Clone()
	})
	logConfig := NewConfig(&DefaultLevels, AllLevelsActive, config)

	timestamp, _ := time.Parse("2006-01-02 15:04:05", "2006-01-02 15:04:05")
	writer := config.WriterForNewMessage(context.Background(), DefaultLevels.Info)
	writer.BeginMessage(logConfig, timestamp, DefaultLevels.Info, "", "test")
	writer.WriteKey("error")
	writer.WriteError(nil)
	writer.CommitMessage()

	if len(gotAttribs) != 1 {
		t.Fatalf("expected 1 attrib, got %d", len(gotAttribs))
	}
	if _, ok := gotAttribs[0].(*Nil); !ok {
		t.Errorf("expected *Nil attrib, got %T", gotAttribs[0])
	}
}

func ExampleCallbackWriter() {
	config := NewConfig(
		&DefaultLevels,
		AllLevelsActive,
		NewCallbackWriterConfig(func(timestamp time.Time, level Level, prefix, text string, attribs Attribs) {
			fmt.Printf("%s|%s|%s: %s", timestamp.Format("2006-01-02 15:04:05"), DefaultLevels.Name(level), prefix, text)
			for _, attrib := range attribs {
				fmt.Printf(" %s", attrib)
			}
			fmt.Println()
		}),
	)
	log := NewLogger(config).WithPrefix("test")

	// Use fixed timestamp for reproducable example output
	timestamp, _ := time.Parse("2006-01-02 15:04:05", "2006-01-02 15:04:05")

	log.InfoAt(timestamp, "My log message").
		Int("int", 66).
		Str("str", "Hello\tWorld!\n").
		Log()

	log.ErrorAt(timestamp, "This is an error").
		Err(errors.New("test error")).
		Log()

	log = log.With().
		Str("subLoggerAttrib", "original").
		SubLogger()

	log.DebugAt(timestamp, "Don't overwrite subLoggerAttrib").
		Str("subLoggerAttrib", "overwritten").
		Log()

	// Output:
	// 2006-01-02 15:04:05|INFO|test: My log message Int{"int": 66} String{"str": "Hello\tWorld!\n"}
	// 2006-01-02 15:04:05|ERROR|test: This is an error Error{"error": "test error"}
	// 2006-01-02 15:04:05|DEBUG|test: Don't overwrite subLoggerAttrib String{"subLoggerAttrib": "original"}
}

// TestCallbackWriter_slicesWithoutSliceAttrib tests slices that create no
// typed slice attrib: an empty slice and slices with JSON or nil elements.
// WriteSliceEnd used to append a nil attrib for an empty slice, which made
// the callback and Attribs.Free panic, and JSON and nil elements were passed
// as separate attribs with the slice key, losing their position in the slice.
func TestCallbackWriter_slicesWithoutSliceAttrib(t *testing.T) {
	var got []Attrib
	config := NewCallbackWriterConfig(func(timestamp time.Time, level Level, prefix, text string, attribs Attribs) {
		for _, a := range attribs {
			got = append(got, a.Clone())
		}
	})
	log := NewLogger(NewConfig(&DefaultLevels, AllLevelsActive, config))

	for _, tc := range []struct {
		name string
		val  any
		json string
	}{
		{name: "empty", val: []int{}, json: `"k":[]`},
		{name: "JSON", val: []json.RawMessage{json.RawMessage(`{"a":1}`), json.RawMessage(`2`)}, json: `"k":[{"a":1},2]`},
		{name: "Loggable between strings", val: []any{"x", Attribs{NewString("s", "v")}, "y"}, json: `"k":["x",{"s":"v"},"y"]`},
		{name: "nil between ints", val: []any{1, nil, 2}, json: `"k":[1,null,2]`},
		{name: "only nil", val: []any{nil}, json: `"k":[null]`},
		// A nil element turns a typed slice into a mixed slice,
		// whose other elements are still written like Message.Any does
		{name: "errors with nil", val: []error{nil, errors.New("boom")}, json: `"k":[null,"boom"]`},
		{name: "UUIDs with nil", val: [][16]byte{{}, {0xa5, 0x47}}, json: `"k":[null,"a5470000-0000-0000-0000-000000000000"]`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got = nil
			log.Info("msg").Any("k", tc.val).Log()
			if len(got) != 1 {
				t.Fatalf("got %d attribs, want one for the slice: %v", len(got), got)
			}
			if j := string(got[0].AppendJSON(nil)); j != tc.json {
				t.Errorf("got %s, want %s", j, tc.json)
			}
		})
	}
}

// TestCallbackWriter_mixedSlice tests a []any with elements of different
// types. Message.Any writes every element with its own type, and the
// CallbackWriter used to replace its typed slice attrib whenever the type
// changed, so the callback only got the last elements of the same type.
func TestCallbackWriter_mixedSlice(t *testing.T) {
	var got []Attrib
	config := NewCallbackWriterConfig(func(timestamp time.Time, level Level, prefix, text string, attribs Attribs) {
		for _, a := range attribs {
			got = append(got, a.Clone())
		}
	})
	log := NewLogger(NewConfig(&DefaultLevels, AllLevelsActive, config))

	log.Info("mixed").Any("k", []any{1, 2, "a", true, 1.5}).Log()
	if len(got) != 1 {
		t.Fatalf("got %d attribs, want 1: %v", len(got), got)
	}
	want := []any{int64(1), int64(2), "a", true, 1.5}
	if fmt.Sprint(got[0].Value()) != fmt.Sprint(want) || got[0].Key() != "k" {
		t.Errorf("got %s, want all elements %v", got[0], want)
	}

	// A slice of one element type still gets its typed slice attrib
	got = nil
	log.Info("same type").Any("k", []any{1, 2}).Log()
	if want := "Ints{\"k\": []int64{1, 2}}"; len(got) != 1 || got[0].String() != want {
		t.Errorf("got %v, want %s", got, want)
	}
}
