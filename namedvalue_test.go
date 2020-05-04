package golog

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestMergeNamedValues(t *testing.T) {
	makeNamedValues := func(names ...string) (nv []NamedValue) {
		for _, name := range names {
			nv = append(nv, &StringNamedValue{Key: name, Val: name})
		}
		return nv
	}

	comparer := cmp.Comparer(func(a, b NamedValue) bool {
		av, ok := a.(*StringNamedValue)
		if !ok {
			return false
		}
		bv, ok := a.(*StringNamedValue)
		if !ok {
			return false
		}
		return *av == *bv
	})

	type args struct {
		a []NamedValue
		b []NamedValue
	}
	tests := []struct {
		name string
		args args
		want []NamedValue
	}{
		{"nil / nil", args{a: nil, b: nil}, nil},
		{"empty / empty", args{a: []NamedValue{}, b: []NamedValue{}}, nil},
		{"nil / 1", args{a: nil, b: makeNamedValues("1")}, makeNamedValues("1")},
		{"1 / nil", args{a: makeNamedValues("1"), b: nil}, makeNamedValues("1")},
		{"1 / 2", args{a: makeNamedValues("1"), b: makeNamedValues("2")}, makeNamedValues("1", "2")},
		{"1 2 / 1", args{a: makeNamedValues("1", "2"), b: makeNamedValues("2")}, makeNamedValues("1", "2")},
		{"1 / 1 2", args{a: makeNamedValues("1"), b: makeNamedValues("1", "2")}, makeNamedValues("1", "2")},
		{"1 2 3 / 1", args{a: makeNamedValues("1", "2", "3"), b: makeNamedValues("1")}, makeNamedValues("1", "2", "3")},
		{"1 2 3 / 2", args{a: makeNamedValues("1", "2", "3"), b: makeNamedValues("2")}, makeNamedValues("1", "2", "3")},
		{"1 2 3 / 3", args{a: makeNamedValues("1", "2", "3"), b: makeNamedValues("3")}, makeNamedValues("1", "2", "3")},
		{"1 / 1 2 3", args{a: makeNamedValues("1", "2", "3"), b: makeNamedValues("1")}, makeNamedValues("1", "2", "3")},
		{"2 / 1 2 3", args{a: makeNamedValues("1", "2", "3"), b: makeNamedValues("2")}, makeNamedValues("1", "2", "3")},
		{"3 / 1 2 3", args{a: makeNamedValues("1", "2", "3"), b: makeNamedValues("3")}, makeNamedValues("1", "2", "3")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MergeNamedValues(tt.args.a, tt.args.b); !cmp.Equal(got, tt.want, comparer) {
				t.Errorf("MergeNamedValues() = %v, want %v", got, tt.want)
			}
		})
	}
}
