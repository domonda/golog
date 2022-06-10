package golog

import (
	"testing"
)

func TestMergeNamedValues(t *testing.T) {
	stringVals := func(valPrefix string, keys ...string) (nv Values) {
		for _, key := range keys {
			nv = append(nv, &StringValue{Key: key, Val: valPrefix + key})
		}
		return nv
	}
	mergedStringVals := func(keyVals ...string) (nv Values) {
		for i := 0; i < len(keyVals); i += 2 {
			nv = append(nv, &StringValue{Key: keyVals[i], Val: keyVals[i+1]})
		}
		return nv
	}

	stringValsEqual := func(a, b Values) bool {
		if len(a) != len(b) {
			return false
		}
		for i := range a {
			av, ok := a[i].(*StringValue)
			if !ok {
				return false
			}
			bv, ok := b[i].(*StringValue)
			if !ok {
				return false
			}
			if *av != *bv {
				return false
			}
		}
		return true
	}

	type args struct {
		a Values
		b Values
	}
	tests := []struct {
		name string
		args args
		want Values
	}{
		{name: "nil / nil", args: args{a: nil, b: nil}, want: nil},
		{name: "empty / empty", args: args{a: Values{}, b: Values{}}, want: Values{}},
		{name: "nil / 1", args: args{a: nil, b: stringVals("b", "1")}, want: stringVals("b", "1")},
		{name: "1 / nil", args: args{a: stringVals("a", "1"), b: nil}, want: stringVals("a", "1")},
		{name: "1 / 2", args: args{a: stringVals("a", "1"), b: stringVals("b", "2")}, want: mergedStringVals("1", "a1", "2", "b2")},
		{name: "1 2 / 1", args: args{a: stringVals("a", "1", "2"), b: stringVals("b", "2")}, want: mergedStringVals("1", "a1", "2", "b2")},
		{name: "1 / 1 2", args: args{a: stringVals("a", "1"), b: stringVals("b", "1", "2")}, want: mergedStringVals("1", "b1", "2", "b2")},
		{name: "1 2 3 / 1", args: args{a: stringVals("a", "1", "2", "3"), b: stringVals("b", "1")}, want: mergedStringVals("2", "a2", "3", "a3", "1", "b1")},
		{name: "1 2 3 / 2", args: args{a: stringVals("a", "1", "2", "3"), b: stringVals("b", "2")}, want: mergedStringVals("1", "a1", "3", "a3", "2", "b2")},
		{name: "1 2 3 / 3", args: args{a: stringVals("a", "1", "2", "3"), b: stringVals("b", "3")}, want: mergedStringVals("1", "a1", "2", "a2", "3", "b3")},
		{name: "1 / 1 2 3", args: args{a: stringVals("a", "1"), b: stringVals("b", "1", "2", "3")}, want: mergedStringVals("1", "b1", "2", "b2", "3", "b3")},
		{name: "2 / 1 2 3", args: args{a: stringVals("a", "2"), b: stringVals("b", "1", "2", "3")}, want: mergedStringVals("1", "b1", "2", "b2", "3", "b3")},
		{name: "3 / 1 2 3", args: args{a: stringVals("a", "3"), b: stringVals("b", "1", "2", "3")}, want: mergedStringVals("1", "b1", "2", "b2", "3", "b3")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MergeValues(tt.args.a, tt.args.b)
			if !stringValsEqual(got, tt.want) {
				t.Errorf("MergeNamedValues() = %v, want %v", got, tt.want)
			}
		})
	}
}
