package mempool

import "sync"

// Pointer provides a memory pool for pointers to type T to reduce allocations
// in high-frequency logging operations. Pointers are automatically zeroed
// when returned to the pool to prevent memory leaks and data contamination.
//
// This is particularly useful for pooling struct pointers that are frequently
// allocated and deallocated in logging operations.
//
// Example usage:
//
//	pool := &mempool.Pointer[MyStruct]{}
//	ptr := pool.GetOrNew()
//	ptr.Field = "value"
//	pool.ClearAndPutBack(ptr)
type Pointer[T any] struct {
	// pool stores reusable pointer instances
	pool sync.Pool
}

// GetOrNew returns a pointer to T from the pool or creates a new one if the pool is empty.
// The returned pointer is guaranteed to be non-nil and ready for use.
//
// If a pointer is available in the pool, it is returned (the pointed-to value
// may contain data from previous usage, so it should be zeroed before use if needed).
// If the pool is empty, a new T is allocated and a pointer to it is returned.
//
// Returns a *T that is ready for immediate use.
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

// ClearAndPutBack zeroes the pointed-to value and returns the pointer to the pool for reuse.
// The value pointed to by ptr is set to the zero value of type T, then the pointer
// is returned to the pool.
//
// This method is safe to call with nil pointers (no-op).
// After calling this method, the pointer should not be used as it may be
// returned to other goroutines.
//
// Parameters:
//   - ptr: the pointer to zero and return to the pool
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

// Drain removes all pointers from the pool and returns the count of drained items.
// This is useful for testing, cleanup, or when you need to ensure the pool
// is completely empty.
//
// The drained pointers are not zeroed before being discarded, so this method
// should only be called when you're sure the pointers are no longer needed.
//
// Returns the number of pointers that were in the pool.
func (p *Pointer[T]) Drain() int {
	count := 0
	for p.pool.Get() != nil {
		count++
	}
	return count
}
