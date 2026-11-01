package mempool

import (
	"io"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMap(t *testing.T) {
	RegisterCallbacksWriterForTest(t, io.Discard)

	var pool Map[string, any]
	m := pool.GetOrMake()
	require.NotNil(t, m)
	require.Empty(t, m)
	m["key"] = "value"
	require.Equal(t, 1, NumOutstanding())

	pool.ClearAndPutBack(m)
	require.Zero(t, NumOutstanding())
	require.Empty(t, m, "map was cleared")

	reused := pool.GetOrMake()
	require.Empty(t, reused, "reused map is empty")
	require.Equal(t, reflect.ValueOf(m).Pointer(), reflect.ValueOf(reused).Pointer(), "reused the pooled map")
	pool.ClearAndPutBack(reused)
	require.Equal(t, 1, pool.Drain())
}
