package mempool

import "sync"

type Pointer[T any] struct {
	pool sync.Pool
}

// GetOrNew returns a non-nil pointer to T from the pool
// or allocates a new T if the pool is empty.
func (p *Pointer[T]) GetOrNew() *T {
	pointer, ok := p.pool.Get().(*T)
	if !ok {
		pointer = new(T)
	}
	if onPointerGetOrNew != nil {
		onPointerGetOrNew(pointer, ok)
	}
	return pointer
}

// ClearAndPutBack sets the pointed to value to the zero value and returns
// the pointer to the pool.
func (p *Pointer[T]) ClearAndPutBack(ptr *T) {
	if onPutBack != nil {
		onPutBack(ptr)
	}
	if ptr != nil {
		var zero T
		*ptr = zero
		p.pool.Put(ptr)
	}
}

// Drain empties the pool and returns the number of items drained.
func (p *Pointer[T]) Drain() int {
	count := 0
	for p.pool.Get() != nil {
		count++
	}
	return count
}
