package golog

import (
	"context"
	"testing"
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
