package mempool

import (
	"fmt"
	"io"
	"testing"
)

var (
	onPointerGetOrNew func(value any, reused bool)
	onSliceGetOrMake  func(value any, reused bool, length, capacity int)
	onMapGetOrMake    func(value any, reused bool, capacity int)
	onPutBack         func(value any)

	numOutstanding map[string]int
)

func RegisterCallbacksWriterForTest(t *testing.T, w io.Writer) {
	t.Helper()

	t.Cleanup(func() {
		onPointerGetOrNew = nil
		onSliceGetOrMake = nil
		onMapGetOrMake = nil
		onPutBack = nil
		numOutstanding = nil
	})

	numOutstanding = make(map[string]int)
	onPointerGetOrNew = func(value any, reused bool) {
		if reused {
			fmt.Fprintf(w, "Reused %T\n", value)
		} else {
			fmt.Fprintf(w, "Allocated %T\n", value)
		}
		numOutstanding[fmt.Sprintf("%T", value)]++
	}
	onSliceGetOrMake = func(value any, reused bool, length, capacity int) {
		if reused {
			fmt.Fprintf(w, "Reused %T len:%d cap:%d\n", value, length, capacity)
		} else {
			fmt.Fprintf(w, "Allocated %T len:%d cap:%d\n", value, length, capacity)
		}
		numOutstanding[fmt.Sprintf("%T", value)]++
	}
	onMapGetOrMake = func(value any, reused bool, capacity int) {
		if reused {
			fmt.Fprintf(w, "Reused %T cap:%d\n", value, capacity)
		} else {
			fmt.Fprintf(w, "Allocated %T cap:%d\n", value, capacity)
		}
		numOutstanding[fmt.Sprintf("%T", value)]++
	}
	onPutBack = func(value any) {
		fmt.Fprintf(w, "Returned %T\n", value)
		numOutstanding[fmt.Sprintf("%T", value)]--
	}
}

func AssertNoOutstanding(t *testing.T) {
	t.Helper()

	for typ, num := range numOutstanding {
		if num != 0 {
			t.Errorf("%d outstanding %s", num, typ)
		}
	}
}
