package golog

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestMergeNamedValues(t *testing.T) {
	makeNamedValues := func(names ...string) (nv Values) {
		for _, name := range names {
			nv = append(nv, &StringValue{Key: name, Val: name})
		}
		return nv
	}

	comparer := cmp.Comparer(func(a, b Value) bool {
		av, ok := a.(*StringValue)
		if !ok {
			return false
		}
		bv, ok := a.(*StringValue)
		if !ok {
			return false
		}
		return *av == *bv
	})

	type args struct {
		a Values
		b Values
	}
	tests := []struct {
		name string
		args args
		want Values
	}{
		{"nil / nil", args{a: nil, b: nil}, nil},
		{"empty / empty", args{a: Values{}, b: Values{}}, nil},
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
			if got := MergeValues(tt.args.a, tt.args.b); !cmp.Equal(got, tt.want, comparer) {
				t.Errorf("MergeNamedValues() = %v, want %v", got, tt.want)
			}
		})
	}
}
