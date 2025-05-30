package mempool

import "sync"

type Slice[T any] struct {
	pool   sync.Pool
	MinCap int
}

// GetOrMake returns a slice with the passed length from the pool
// or a new slice if the pool is empty or the capacity of the
// first slice from the pool is too small.
// The passed capacity is used only when allocating a new slice.
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

// ClearAndPutBack clears the slice and returns it to the pool.
func (p *Slice[T]) ClearAndPutBack(slice []T) {
	if onPutBack != nil {
		onPutBack(slice)
	}
	if slice != nil {
		clear(slice[:cap(slice)]) // Clear complete capacity of the slice
		p.pool.Put(slice[:0])
	}
}

// Drain empties the pool and returns the number of items drained.
func (p *Slice[T]) Drain() int {
	count := 0
	for p.pool.Get() != nil {
		count++
	}
	return count
}
