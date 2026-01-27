package golog

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_uniqueWriterConfigs(t *testing.T) {
	var (
		c0 = NopWriterConfig("c0")
		c1 = NopWriterConfig("c1")
		c2 = NopWriterConfig("c2")
	)
	tests := []struct {
		name       string
		w          []WriterConfig
		wantUnique []WriterConfig
	}{
		{name: "empty", w: nil, wantUnique: nil},
		{name: "nil only", w: []WriterConfig{nil, nil, nil}, wantUnique: nil},
		{name: "single", w: []WriterConfig{c0}, wantUnique: []WriterConfig{c0}},
		{name: "two different", w: []WriterConfig{c0, c1}, wantUnique: []WriterConfig{c0, c1}},
		{name: "three different", w: []WriterConfig{c0, c1, c2}, wantUnique: []WriterConfig{c0, c1, c2}},
		{name: "duplicate", w: []WriterConfig{c0, c0}, wantUnique: []WriterConfig{c0}},
		{name: "triplicate", w: []WriterConfig{c0, c0, c0}, wantUnique: []WriterConfig{c0}},
		{name: "nil at beginning", w: []WriterConfig{nil, c0, c1}, wantUnique: []WriterConfig{c0, c1}},
		{name: "nil in between", w: []WriterConfig{c0, nil, c1}, wantUnique: []WriterConfig{c0, c1}},
		{name: "nil at end", w: []WriterConfig{c0, c1, nil}, wantUnique: []WriterConfig{c0, c1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotUnique := uniqueNonNilWriterConfigs(tt.w)
			require.Equal(t, tt.wantUnique, gotUnique)
		})
	}
}
