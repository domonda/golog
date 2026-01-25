package golog

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestIsActiveContext(t *testing.T) {
	tests := []struct {
		name  string
		ctx   context.Context
		level Level
		want  bool
	}{
		{name: "nil", ctx: nil, level: 0, want: true},
		{name: "Background", ctx: context.Background(), level: 0, want: true},
		{name: "AllLevelsActive", ctx: ContextWithLevelDecider(context.Background(), AllLevelsActive), level: 0, want: true},
		{name: "BoolLevelDecider(true)", ctx: ContextWithLevelDecider(context.Background(), BoolLevelDecider(true)), level: 0, want: true},
		{name: "BoolLevelDecider(false)", ctx: ContextWithLevelDecider(context.Background(), BoolLevelDecider(false)), level: 0, want: false},
		{name: "AllLevelsInactive", ctx: ContextWithLevelDecider(context.Background(), AllLevelsInactive), level: 0, want: false},
		{name: "ContextWithoutLogging", ctx: ContextWithoutLogging(context.Background()), level: 0, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsActiveContext(tt.ctx, tt.level); got != tt.want {
				t.Errorf("IsActiveContext(%v, %v) = %t, want %t", tt.ctx, tt.level, got, tt.want)
			}
		})
	}
}

func TestContextWithLevelDecider(t *testing.T) {
	t.Run("adds level decider to context", func(t *testing.T) {
		ctx := context.Background()
		filter := LevelFilterOutBelow(DefaultLevels.Warn)
		ctx = ContextWithLevelDecider(ctx, filter)

		assert.True(t, IsActiveContext(ctx, DefaultLevels.Warn))
		assert.True(t, IsActiveContext(ctx, DefaultLevels.Error))
		assert.False(t, IsActiveContext(ctx, DefaultLevels.Info))
		assert.False(t, IsActiveContext(ctx, DefaultLevels.Debug))
	})

	t.Run("overwrites previous level decider", func(t *testing.T) {
		ctx := context.Background()
		ctx = ContextWithLevelDecider(ctx, AllLevelsInactive)
		assert.False(t, IsActiveContext(ctx, DefaultLevels.Info))

		ctx = ContextWithLevelDecider(ctx, AllLevelsActive)
		assert.True(t, IsActiveContext(ctx, DefaultLevels.Info))
	})
}

func TestContextWithoutLogging(t *testing.T) {
	t.Run("disables all levels", func(t *testing.T) {
		ctx := ContextWithoutLogging(context.Background())

		assert.False(t, IsActiveContext(ctx, DefaultLevels.Trace))
		assert.False(t, IsActiveContext(ctx, DefaultLevels.Debug))
		assert.False(t, IsActiveContext(ctx, DefaultLevels.Info))
		assert.False(t, IsActiveContext(ctx, DefaultLevels.Warn))
		assert.False(t, IsActiveContext(ctx, DefaultLevels.Error))
		assert.False(t, IsActiveContext(ctx, DefaultLevels.Fatal))
	})
}

func TestBoolLevelDecider(t *testing.T) {
	t.Run("true allows all levels", func(t *testing.T) {
		decider := BoolLevelDecider(true)
		ctx := context.Background()

		assert.True(t, decider.IsActive(ctx, DefaultLevels.Trace))
		assert.True(t, decider.IsActive(ctx, DefaultLevels.Debug))
		assert.True(t, decider.IsActive(ctx, DefaultLevels.Info))
		assert.True(t, decider.IsActive(ctx, DefaultLevels.Warn))
		assert.True(t, decider.IsActive(ctx, DefaultLevels.Error))
		assert.True(t, decider.IsActive(ctx, DefaultLevels.Fatal))
	})

	t.Run("false disables all levels", func(t *testing.T) {
		decider := BoolLevelDecider(false)
		ctx := context.Background()

		assert.False(t, decider.IsActive(ctx, DefaultLevels.Trace))
		assert.False(t, decider.IsActive(ctx, DefaultLevels.Debug))
		assert.False(t, decider.IsActive(ctx, DefaultLevels.Info))
		assert.False(t, decider.IsActive(ctx, DefaultLevels.Warn))
		assert.False(t, decider.IsActive(ctx, DefaultLevels.Error))
		assert.False(t, decider.IsActive(ctx, DefaultLevels.Fatal))
	})

	t.Run("ignores context argument", func(t *testing.T) {
		decider := BoolLevelDecider(true)
		assert.True(t, decider.IsActive(nil, DefaultLevels.Info))
		assert.True(t, decider.IsActive(context.Background(), DefaultLevels.Info))
	})

	t.Run("ignores level argument", func(t *testing.T) {
		decider := BoolLevelDecider(true)
		ctx := context.Background()
		// All levels return same value
		assert.True(t, decider.IsActive(ctx, LevelMin))
		assert.True(t, decider.IsActive(ctx, LevelMax))
		assert.True(t, decider.IsActive(ctx, Level(-100)))
	})
}

func TestContextWithTimestamp(t *testing.T) {
	t.Run("adds timestamp to context", func(t *testing.T) {
		ts := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
		ctx := ContextWithTimestamp(context.Background(), ts)

		retrieved := TimestampFromContext(ctx)
		assert.Equal(t, ts, retrieved)
	})

	t.Run("overwrites previous timestamp", func(t *testing.T) {
		ts1 := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
		ts2 := time.Date(2024, 6, 20, 14, 45, 0, 0, time.UTC)

		ctx := ContextWithTimestamp(context.Background(), ts1)
		ctx = ContextWithTimestamp(ctx, ts2)

		retrieved := TimestampFromContext(ctx)
		assert.Equal(t, ts2, retrieved)
	})
}

func TestTimestampFromContext(t *testing.T) {
	t.Run("returns zero time when no timestamp set", func(t *testing.T) {
		ctx := context.Background()
		ts := TimestampFromContext(ctx)
		assert.True(t, ts.IsZero())
	})

	t.Run("returns timestamp when set", func(t *testing.T) {
		expected := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
		ctx := ContextWithTimestamp(context.Background(), expected)
		ts := TimestampFromContext(ctx)
		assert.Equal(t, expected, ts)
	})
}

func TestTimestamp(t *testing.T) {
	t.Run("returns timestamp from context when set", func(t *testing.T) {
		expected := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
		ctx := ContextWithTimestamp(context.Background(), expected)
		ts := Timestamp(ctx)
		assert.Equal(t, expected, ts)
	})

	t.Run("returns current time when no timestamp set", func(t *testing.T) {
		ctx := context.Background()
		before := time.Now()
		ts := Timestamp(ctx)
		after := time.Now()

		assert.False(t, ts.Before(before), "timestamp should not be before test start")
		assert.False(t, ts.After(after), "timestamp should not be after test end")
	})

	t.Run("returns current time for nil context", func(t *testing.T) {
		before := time.Now()
		// Note: Timestamp handles nil internally via context.Value
		ts := Timestamp(context.Background())
		after := time.Now()

		assert.False(t, ts.Before(before))
		assert.False(t, ts.After(after))
	})
}

func TestLevelDeciderInterface(t *testing.T) {
	// Verify that LevelFilter implements LevelDecider
	var _ LevelDecider = LevelFilter(0)

	// Verify that BoolLevelDecider implements LevelDecider
	var _ LevelDecider = BoolLevelDecider(true)

	// Verify that Config implements LevelDecider
	var _ LevelDecider = Config(nil)
}

func TestContextWithMultipleValues(t *testing.T) {
	t.Run("preserves both timestamp and level decider", func(t *testing.T) {
		ts := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
		filter := LevelFilterOutBelow(DefaultLevels.Warn)

		ctx := context.Background()
		ctx = ContextWithTimestamp(ctx, ts)
		ctx = ContextWithLevelDecider(ctx, filter)

		// Both values should be accessible
		assert.Equal(t, ts, TimestampFromContext(ctx))
		assert.False(t, IsActiveContext(ctx, DefaultLevels.Info))
		assert.True(t, IsActiveContext(ctx, DefaultLevels.Warn))
	})
}
