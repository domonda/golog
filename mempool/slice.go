package mempool

import "sync"

// Slice provides a memory pool for []T slices to reduce allocations
// in high-frequency logging operations. Slices are automatically cleared
// when returned to the pool to prevent memory leaks and data contamination.
//
// The pool maintains a minimum capacity (MinCap) for newly allocated slices
// and can reuse slices with sufficient capacity, even if they're smaller
// than the requested capacity.
//
// Example usage:
//
//	pool := &mempool.Slice[string]{MinCap: 16}
//	slice := pool.GetOrMake(5, 20) // length=5, capacity=20
//	slice[0] = "value"
//	pool.ClearAndPutBack(slice)
type Slice[T any] struct {
	// pool stores reusable slice instances
	pool sync.Pool
	// MinCap is the minimum capacity for newly allocated slices
	MinCap int
}

// GetOrMake returns a slice with the specified length from the pool or creates a new one.
// The returned slice is guaranteed to be non-nil and ready for use.
//
// If a slice with sufficient capacity is available in the pool, it is reused
// and resized to the requested length. If the pool is empty or the first
// available slice has insufficient capacity, a new slice is allocated.
//
// The capacity parameter is only used when allocating a new slice. The actual
// capacity of the returned slice may be larger than requested due to MinCap.
//
// Parameters:
//   - length: the desired length of the slice
//   - capacity: the desired capacity (only used for new allocations)
//
// Returns a []T with the specified length, ready for immediate use.
func (p *Slice[T]) GetOrMake(length, capacity int) []T {
	slice, ok := p.pool.Get().([]T)
	if ok {
		if length <= cap(slice) {
			// cap(slice) might be smaller than the passed capacity,
			// but we can live with that
			slice = slice[:length]
			if onSliceGetOrMake != nil {
				onSliceGetOrMake(slice, true, length, capacity)
			}
			return slice
		}
		// The capacity of the slice from the pool is too small,
		// so put it back and allocate a new one.
		p.pool.Put(slice)
	}
	slice = make([]T, length, max(capacity, p.MinCap))
	if onSliceGetOrMake != nil {
		onSliceGetOrMake(slice, false, length, max(capacity, p.MinCap))
	}
	return slice
}

// ClearAndPutBack clears the slice and returns it to the pool for reuse.
// The slice is cleared using Go's built-in clear() function to zero all
// elements in the slice's capacity, then returned to the pool.
//
// This method is safe to call with nil slices (no-op).
// After calling this method, the slice should not be used as it may be
// returned to other goroutines.
//
// Parameters:
//   - slice: the slice to clear and return to the pool
func (p *Slice[T]) ClearAndPutBack(slice []T) {
	if onPutBack != nil {
		onPutBack(slice)
	}
	if slice != nil {
		clear(slice[:cap(slice)]) // Clear complete capacity of the slice
		p.pool.Put(slice[:0])
	}
}

// Drain removes all slices from the pool and returns the count of drained items.
// This is useful for testing, cleanup, or when you need to ensure the pool
// is completely empty.
//
// The drained slices are not cleared before being discarded, so this method
// should only be called when you're sure the slices are no longer needed.
//
// Returns the number of slices that were in the pool.
func (p *Slice[T]) Drain() int {
	count := 0
	for p.pool.Get() != nil {
		count++
	}
	return count
}
