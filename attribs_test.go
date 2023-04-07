package golog

import (
	"testing"
)

func TestMergeAttribs(t *testing.T) {
	stringVals := func(valPrefix string, keys ...string) (nv Attribs) {
		for _, key := range keys {
			nv = append(nv, &String{Key: key, Val: valPrefix + key})
		}
		return nv
	}
	mergedStringVals := func(keyVals ...string) (nv Attribs) {
		for i := 0; i < len(keyVals); i += 2 {
			nv = append(nv, &String{Key: keyVals[i], Val: keyVals[i+1]})
		}
		return nv
	}

	stringValsEqual := func(a, b Attribs) bool {
		if len(a) != len(b) {
			return false
		}
		for i := range a {
			av, ok := a[i].(*String)
			if !ok {
				return false
			}
			bv, ok := b[i].(*String)
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
		a Attribs
		b Attribs
	}
	tests := []struct {
		name string
		args args
		want Attribs
	}{
		{name: "nil / nil", args: args{a: nil, b: nil}, want: nil},
		{name: "empty / empty", args: args{a: Attribs{}, b: Attribs{}}, want: Attribs{}},
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

		{name: "nil / Values{nil}", args: args{a: nil, b: Attribs{nil}}, want: nil},
		{name: "Values{nil} / nil", args: args{a: Attribs{nil}, b: nil}, want: nil},
		{name: "Values{nil} / Values{nil, nil}", args: args{a: Attribs{nil}, b: Attribs{nil, nil}}, want: nil},
		{name: "1 / 2 nil", args: args{a: stringVals("a", "1"), b: append(stringVals("b", "2"), nil)}, want: mergedStringVals("1", "a1", "2", "b2")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MergeAttribs(tt.args.a, tt.args.b)
			if !stringValsEqual(got, tt.want) {
				t.Errorf("MergeAttribs() = %v, want %v", got, tt.want)
			}
		})
	}
}
