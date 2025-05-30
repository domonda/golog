package mempool

import "sync"

type Map[K comparable, V any] struct {
	pool   sync.Pool
	MinCap int
}

// GetOrMake returns a map from the pool
// or makes a new one with p.MinCap capacity.
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

// ClearAndPutBack clears the map and returns it to the pool.
func (p *Map[K, V]) ClearAndPutBack(m map[K]V) {
	if onPutBack != nil {
		onPutBack(m)
	}
	if m != nil {
		clear(m)
		p.pool.Put(m)
	}
}

// Drain empties the pool and returns the number of items drained.
func (p *Map[K, V]) Drain() int {
	count := 0
	for p.pool.Get() != nil {
		count++
	}
	return count
}
