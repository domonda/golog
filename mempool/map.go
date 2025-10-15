package mempool

import "sync"

// Map provides a memory pool for map[K]V instances to reduce allocations
// in high-frequency logging operations. Maps are automatically cleared
// when returned to the pool to prevent memory leaks.
//
// The pool maintains a minimum capacity (MinCap) for newly allocated maps
// to reduce the need for map growth during usage.
//
// Example usage:
//
//	pool := &mempool.Map[string, any]{MinCap: 16}
//	m := pool.GetOrMake()
//	m["key"] = "value"
//	pool.ClearAndPutBack(m)
type Map[K comparable, V any] struct {
	// pool stores reusable map instances
	pool sync.Pool
	// MinCap is the minimum capacity for newly allocated maps
	MinCap int
}

// GetOrMake returns a map from the pool or creates a new one if the pool is empty.
// The returned map is guaranteed to be non-nil and ready for use.
//
// If a map is available in the pool, it is returned (and may contain data from
// previous usage, so it should be cleared before use if needed).
// If the pool is empty, a new map is created with MinCap capacity.
//
// Returns a map[K]V that is ready for immediate use.
func (p *Map[K, V]) GetOrMake() map[K]V {
	m, ok := p.pool.Get().(map[K]V)
	if !ok {
		m = make(map[K]V, p.MinCap)
	}
	if onMapGetOrMake != nil {
		onMapGetOrMake(m, ok, p.MinCap)
	}
	return m
}

// ClearAndPutBack clears the map and returns it to the pool for reuse.
// The map is cleared using Go's built-in clear() function to remove all
// key-value pairs, then returned to the pool.
//
// This method is safe to call with nil maps (no-op).
// After calling this method, the map should not be used as it may be
// returned to other goroutines.
//
// Parameters:
//   - m: the map to clear and return to the pool
func (p *Map[K, V]) ClearAndPutBack(m map[K]V) {
	if onPutBack != nil {
		onPutBack(m)
	}
	if m != nil {
		clear(m)
		p.pool.Put(m)
	}
}

// Drain removes all maps from the pool and returns the count of drained items.
// This is useful for testing, cleanup, or when you need to ensure the pool
// is completely empty.
//
// The drained maps are not cleared before being discarded, so this method
// should only be called when you're sure the maps are no longer needed.
//
// Returns the number of maps that were in the pool.
func (p *Map[K, V]) Drain() int {
	count := 0
	for p.pool.Get() != nil {
		count++
	}
	return count
}
