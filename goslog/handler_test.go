package goslog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"sync"
	"testing"
	"testing/slogtest"

	"github.com/stretchr/testify/require"

	"github.com/domonda/golog"
)

func TestHandler(t *testing.T) {
	var rec recorder
	config := golog.NewConfig(&golog.DefaultLevels, golog.AllLevelsActive, &rec)

	handler := Handler(golog.NewLogger(config), ConvertDefaultLevels)

	err := slogtest.TestHandler(handler, func() []map[string]any {
		return rec.Result
	})
	require.NoError(t, err)
}

func TestConvertDefaultLevels(t *testing.T) {
	tests := []struct {
		name string
		l    slog.Level
		want golog.Level
	}{
		{name: "Debug", l: slog.LevelDebug, want: golog.DefaultLevels.Debug},
		{name: "Debug-1", l: slog.LevelDebug - 1, want: golog.DefaultLevels.Debug - 1},
		{name: "Info", l: slog.LevelInfo, want: golog.DefaultLevels.Info},
		{name: "Warn", l: slog.LevelWarn, want: golog.DefaultLevels.Warn},
		{name: "Error", l: slog.LevelError, want: golog.DefaultLevels.Error},
		{name: "0", l: 0, want: 0},
		{name: "1000", l: 1000, want: golog.LevelInvalid},
		{name: "-1000", l: -1000, want: golog.LevelInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ConvertDefaultLevels(tt.l); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ConvertDefaultLevels(%d) = %v, want %v", tt.l, got, tt.want)
			}
		})
	}
}

// TestHandler_WithAttrsSiblings checks that handlers derived from the same
// parent don't share the backing array of their attrs, so that a sibling's
// WithAttrs doesn't overwrite the attrs of an earlier sibling.
func TestHandler_WithAttrsSiblings(t *testing.T) {
	var rec recorder
	config := golog.NewConfig(&golog.DefaultLevels, golog.AllLevelsActive, &rec)
	// Three single attr WithAttrs calls grow the attrs to len 3 and cap 4,
	// leaving spare capacity that a shared backing array would hand out twice
	base := slog.New(Handler(golog.NewLogger(config), ConvertDefaultLevels)).
		With("a", 1).
		With("b", 2).
		With("c", 3)

	reqA := base.With("request", "A")
	reqB := base.With("request", "B")
	reqA.Info("from A")
	reqB.Info("from B")

	require.Len(t, rec.Result, 2)
	require.Equal(t, "A", rec.Result[0]["request"])
	require.Equal(t, "B", rec.Result[1]["request"])
}

// TestHandler_Concurrent derives handlers from a shared parent and logs
// with them from many goroutines, for go test -race.
// Every logged line must have the attrs of the goroutine that logged it.
func TestHandler_Concurrent(t *testing.T) {
	const (
		numGoroutines = 16
		numMessages   = 50
	)
	var out syncBuffer
	config := golog.NewConfig(&golog.DefaultLevels, golog.AllLevelsActive, golog.NewJSONWriterConfig(&out, nil))
	base := slog.New(Handler(golog.NewLogger(config), ConvertDefaultLevels)).
		With("a", 1).
		With("b", 2).
		With("c", 3)

	var wg sync.WaitGroup
	for g := range numGoroutines {
		wg.Go(func() {
			logger := base.With("g", g).WithGroup("grp").With("g2", g)
			for i := range numMessages {
				logger.Info("msg", "i", i, "gi", fmt.Sprintf("%d-%d", g, i))
			}
		})
	}
	wg.Wait()

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	require.Len(t, lines, numGoroutines*numMessages)
	for _, line := range lines {
		var m map[string]any
		require.NoError(t, json.Unmarshal([]byte(line), &m), line)
		g, i := m["g"], m["grp.i"]
		require.Equal(t, g, m["grp.g2"], line)
		require.Equal(t, fmt.Sprintf("%v-%v", g, i), m["grp.gi"], line)
		require.Equal(t, float64(3), m["c"], line)
	}
}

// syncBuffer is a bytes.Buffer that is safe for concurrent use.
type syncBuffer struct {
	mtx sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mtx.Lock()
	defer b.mtx.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mtx.Lock()
	defer b.mtx.Unlock()
	return b.buf.String()
}

// panicValuer panics in its LogValue method.
type panicValuer struct{}

func (panicValuer) LogValue() slog.Value { panic("LogValue boom") }

// cyclicValuer logs a group that contains itself.
type cyclicValuer struct{ next *cyclicValuer }

func (v *cyclicValuer) LogValue() slog.Value {
	return slog.GroupValue(slog.Any("next", v.next))
}

// TestHandler_NeverPanicsOrHangs tests that a LogValue method that panics
// is logged as error, and that groups nested in each other without end,
// like a LogValuer whose group contains itself, end with an error
// instead of recursing until the stack overflows.
func TestHandler_NeverPanicsOrHangs(t *testing.T) {
	var rec recorder
	config := golog.NewConfig(&golog.DefaultLevels, golog.AllLevelsActive, &rec)
	logger := slog.New(Handler(golog.NewLogger(config), ConvertDefaultLevels))

	logger.Info("panic", "k", panicValuer{})
	require.Len(t, rec.Result, 1)
	require.Contains(t, fmt.Sprint(rec.Result[0]["k"]), "LogValue panicked")

	cyclic := &cyclicValuer{}
	cyclic.next = cyclic
	logger.Info("cycle", "k", cyclic)
	require.Len(t, rec.Result, 2)
	require.Contains(t, fmt.Sprint(rec.Result[1]), "nested more than 64 deep")
}

// TestHandler_MaxRecordAttrs tests that a record with exactly
// maxRecordAttrs attributes is logged completely, and that only the
// attribute above the limit is replaced by the error.
func TestHandler_MaxRecordAttrs(t *testing.T) {
	var rec recorder
	config := golog.NewConfig(&golog.DefaultLevels, golog.AllLevelsActive, &rec)
	logger := slog.New(Handler(golog.NewLogger(config), ConvertDefaultLevels))

	attrs := make([]any, 0, maxRecordAttrs+2)
	for i := range maxRecordAttrs {
		attrs = append(attrs, slog.Int(fmt.Sprintf("k%d", i), i))
	}
	logger.Info("at limit", attrs...)
	require.Len(t, rec.Result, 1)
	last := fmt.Sprintf("k%d", maxRecordAttrs-1)
	require.Equal(t, int64(maxRecordAttrs-1), rec.Result[0][last], "last attribute at the limit")
	require.NotContains(t, fmt.Sprint(rec.Result[0]), "more than")

	attrs = append(attrs, slog.Int("over", 1), slog.Int("ignored", 2))
	logger.Info("over limit", attrs...)
	require.Len(t, rec.Result, 2)
	require.Equal(t, int64(maxRecordAttrs-1), rec.Result[1][last])
	require.Contains(t, fmt.Sprint(rec.Result[1]["over"]), "more than 10000 attributes")
	require.NotContains(t, rec.Result[1], "ignored")
}
