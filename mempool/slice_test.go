package mempool

import (
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSlice_GetOrMake(t *testing.T) {
	t.Run("new slice has MinCap", func(t *testing.T) {
		pool := Slice[int]{MinCap: 8}
		slice := pool.GetOrMake(2, 4)
		require.Len(t, slice, 2)
		require.Equal(t, 8, cap(slice))
	})

	t.Run("reuses slice with enough capacity", func(t *testing.T) {
		var pool Slice[int]
		pooled := pool.GetOrMake(3, 10)
		pooled[0] = 1
		pool.ClearAndPutBack(pooled)

		slice := pool.GetOrMake(2, 10)
		require.Len(t, slice, 2)
		require.Same(t, &pooled[:1][0], &slice[0], "reused the pooled backing array")
		require.Zero(t, slice[0], "pooled slice was cleared")
	})

	// Appending up to the requested capacity must not allocate
	// a new backing array that would be put back into the pool
	// instead of the one taken from it.
	t.Run("does not reuse slice with too little capacity", func(t *testing.T) {
		var pool Slice[int]
		pool.ClearAndPutBack(make([]int, 0, 4))

		slice := pool.GetOrMake(2, 5)
		require.Len(t, slice, 2)
		require.GreaterOrEqual(t, cap(slice), 5)
	})

	t.Run("capacity smaller than length", func(t *testing.T) {
		var pool Slice[int]
		slice := pool.GetOrMake(3, 1)
		require.Len(t, slice, 3)
		require.GreaterOrEqual(t, cap(slice), 3)
	})
}

func TestSlice_ClearAndPutBack(t *testing.T) {
	RegisterCallbacksWriterForTest(t, io.Discard)

	var pool Slice[int]
	slice := pool.GetOrMake(2, 2)
	require.Equal(t, 1, NumOutstanding())
	pool.ClearAndPutBack(slice)
	require.Zero(t, NumOutstanding())
	require.Equal(t, 1, pool.Drain())
}
