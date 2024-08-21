package golog

import (
	"reflect"
	"testing"
)

func Test_uniqueWriterConfigs(t *testing.T) {
	var (
		c0 = NopWriterConfig("c0")
		c1 = NopWriterConfig("c1")
		c2 = NopWriterConfig("c2")
	)
	tests := []struct {
		name string
		w    []WriterConfig
		want []WriterConfig
	}{
		{name: "empty", w: nil, want: nil},
		{name: "nil only", w: []WriterConfig{nil, nil, nil}, want: nil},
		{name: "single", w: []WriterConfig{c0}, want: []WriterConfig{c0}},
		{name: "two different", w: []WriterConfig{c0, c1}, want: []WriterConfig{c0, c1}},
		{name: "three different", w: []WriterConfig{c0, c1, c2}, want: []WriterConfig{c0, c1, c2}},
		{name: "duplicate", w: []WriterConfig{c0, c0}, want: []WriterConfig{c0}},
		{name: "triplicate", w: []WriterConfig{c0, c0, c0}, want: []WriterConfig{c0}},
		{name: "nil at beginning", w: []WriterConfig{nil, c0, c1}, want: []WriterConfig{c0, c1}},
		{name: "nil in between", w: []WriterConfig{c0, nil, c1}, want: []WriterConfig{c0, c1}},
		{name: "nil at end", w: []WriterConfig{c0, c1, nil}, want: []WriterConfig{c0, c1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := uniqueWriterConfigs(tt.w); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("uniqueWriterConfigs(%#v) = %#v, want %Ev", tt.w, got, tt.want)
			}
		})
	}
}
