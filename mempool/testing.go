package mempool

import (
	"fmt"
	"io"
	"sync"
	"testing"
)

// Global callback functions for testing and debugging mempool behavior.
// These are set by RegisterCallbacksWriterForTest and cleared on test cleanup.
var (
	// onPointerGetOrNew is called when a pointer is retrieved from or allocated for the pool
	onPointerGetOrNew func(value any, reused bool)
	// onSliceGetOrMake is called when a slice is retrieved from or allocated for the pool
	onSliceGetOrMake func(value any, reused bool, length, capacity int)
	// onMapGetOrMake is called when a map is retrieved from or allocated for the pool
	onMapGetOrMake func(value any, reused bool, capacity int)
	// onPutBack is called when any pooled item is returned to its pool
	onPutBack func(value any)

	// numOutstanding tracks the number of outstanding items per type for testing
	// This is not thread-safe and should only be used in single-threaded test contexts
	numOutstanding map[string]int
	// numOutstandingMtx protects numOutstanding from concurrent access
	numOutstandingMtx sync.RWMutex
)

// RegisterCallbacksWriterForTest sets up callback functions for testing mempool behavior
// and writes detailed allocation/deallocation information to the provided writer.
//
// This function is designed for testing purposes and should not be used in production code.
// It registers callbacks that track when items are allocated, reused, and returned to pools,
// providing visibility into mempool behavior during tests.
//
// The callbacks are automatically cleaned up when the test completes.
//
// Parameters:
//   - t: the testing.T instance for cleanup registration
//   - w: the io.Writer to write allocation/deallocation logs to
func RegisterCallbacksWriterForTest(t *testing.T, w io.Writer) {
	t.Helper()

	// Cleanup callbacks and tracking data when test completes
	t.Cleanup(func() {
		numOutstandingMtx.Lock()
		defer numOutstandingMtx.Unlock()

		onPointerGetOrNew = nil
		onSliceGetOrMake = nil
		onMapGetOrMake = nil
		onPutBack = nil
		numOutstanding = nil
	})

	// Initialize tracking data
	numOutstandingMtx.Lock()
	numOutstanding = make(map[string]int)
	numOutstandingMtx.Unlock()

	// Set up callbacks for each pool type
	onPointerGetOrNew = func(value any, reused bool) {
		if reused {
			fmt.Fprintf(w, "Reused %T\n", value)
		} else {
			fmt.Fprintf(w, "Allocated %T\n", value)
		}

		numOutstandingMtx.Lock()
		defer numOutstandingMtx.Unlock()
		numOutstanding[fmt.Sprintf("%T", value)]++
	}
	onSliceGetOrMake = func(value any, reused bool, length, capacity int) {
		if reused {
			fmt.Fprintf(w, "Reused %T len:%d cap:%d\n", value, length, capacity)
		} else {
			fmt.Fprintf(w, "Allocated %T len:%d cap:%d\n", value, length, capacity)
		}

		numOutstandingMtx.Lock()
		defer numOutstandingMtx.Unlock()
		numOutstanding[fmt.Sprintf("%T", value)]++
	}
	onMapGetOrMake = func(value any, reused bool, capacity int) {
		if reused {
			fmt.Fprintf(w, "Reused %T cap:%d\n", value, capacity)
		} else {
			fmt.Fprintf(w, "Allocated %T cap:%d\n", value, capacity)
		}

		numOutstandingMtx.Lock()
		defer numOutstandingMtx.Unlock()
		numOutstanding[fmt.Sprintf("%T", value)]++
	}
	onPutBack = func(value any) {
		fmt.Fprintf(w, "Returned %T\n", value)

		numOutstandingMtx.Lock()
		defer numOutstandingMtx.Unlock()
		numOutstanding[fmt.Sprintf("%T", value)]--
	}
}

// NumOutstanding returns the total number of outstanding mempool items across all pools.
// This is useful for testing to ensure that all mempool items are properly returned
// and no memory leaks occur.
//
// The count can be negative if items are returned multiple times (which indicates
// a bug in the calling code). A positive count indicates items that were allocated
// but not yet returned to their pools.
//
// This function is thread-safe and should only be called from test code.
//
// Returns the total number of outstanding items across all pool types.
func NumOutstanding() int {
	numOutstandingMtx.RLock()
	defer numOutstandingMtx.RUnlock()

	total := 0
	for _, num := range numOutstanding {
		total += num
	}
	return total
}
