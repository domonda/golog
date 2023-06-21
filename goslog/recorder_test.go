package goslog

import (
	"reflect"
	"testing"
)

func Test_splitGroupKeyVal(t *testing.T) {
	tests := []struct {
		key     string
		val     any
		wantKey string
		wantVal any
	}{
		{key: "", val: 1, wantKey: "", wantVal: 1},
		{key: "a", val: 1, wantKey: "a", wantVal: 1},
		{key: "G.a", val: 1, wantKey: "G", wantVal: map[string]any{"a": 1}},
		{key: "G.a.b", val: 1, wantKey: "G", wantVal: map[string]any{"a": map[string]any{"b": 1}}},
		{key: "G.a.b.c", val: 1, wantKey: "G", wantVal: map[string]any{"a": map[string]any{"b": map[string]any{"c": 1}}}},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			gotRootKey, gotRootVal := splitGroupKeyVal(tt.key, tt.val)
			if gotRootKey != tt.wantKey {
				t.Errorf("splitGroupKeyVal() gotRootKey = %v, want %v", gotRootKey, tt.wantKey)
			}
			if !reflect.DeepEqual(gotRootVal, tt.wantVal) {
				t.Errorf("splitGroupKeyVal() gotRootVal = %v, want %v", gotRootVal, tt.wantVal)
			}
		})
	}
}
