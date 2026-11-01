package golog

import (
	"context"
	"io"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/domonda/golog/mempool"
)

func TestAttribs_CloneAndAppendNonExisting(t *testing.T) {
	// stringVals := func(valPrefix string, keys ...string) (nv Attribs) {
	// 	for _, key := range keys {
	// 		nv = append(nv, Int{Key: key, Val: valPrefix + key})
	// 	}
	// 	return nv
	// }
	// mergedStringVals := func(keyVals ...string) (nv Attribs) {
	// 	for i := 0; i < len(keyVals); i += 2 {
	// 		nv = append(nv, String{Key: keyVals[i], Val: keyVals[i+1]})
	// 	}
	// 	return nv
	// }

	intValsEqual := func(left, right Attribs) bool {
		if len(left) != len(right) {
			return false
		}
		for i := range left {
			l, ok := left[i].(*Int)
			if !ok {
				return false
			}
			r, ok := right[i].(*Int)
			if !ok {
				return false
			}
			if l != r {
				return false
			}
		}
		return true
	}

	type args struct {
		left  Attribs
		right Attribs
	}
	tests := []struct {
		name string
		args args
		want Attribs
	}{
		{name: "nil / nil", args: args{left: nil, right: nil}, want: nil},
		{name: "empty / empty", args: args{left: Attribs{}, right: Attribs{}}, want: Attribs{}},
		// {name: "nil / 1", args: args{left: nil, right: stringVals("b", "1")}, want: stringVals("b", "1")},
		// {name: "1 / nil", args: args{left: stringVals("a", "1"), right: nil}, want: stringVals("a", "1")},
		// {name: "1 / 2", args: args{left: stringVals("a", "1"), right: stringVals("b", "2")}, want: mergedStringVals("1", "a1", "2", "b2")},
		// {name: "1 2 / 1", args: args{left: stringVals("a", "1", "2"), right: stringVals("b", "2")}, want: mergedStringVals("1", "a1", "2", "b2")},
		// {name: "1 / 1 2", args: args{left: stringVals("a", "1"), right: stringVals("b", "1", "2")}, want: mergedStringVals("1", "b1", "2", "b2")},
		// {name: "1 2 3 / 1", args: args{left: stringVals("a", "1", "2", "3"), right: stringVals("b", "1")}, want: mergedStringVals("2", "a2", "3", "a3", "1", "b1")},
		// {name: "1 2 3 / 2", args: args{left: stringVals("a", "1", "2", "3"), right: stringVals("b", "2")}, want: mergedStringVals("1", "a1", "3", "a3", "2", "b2")},
		// {name: "1 2 3 / 3", args: args{left: stringVals("a", "1", "2", "3"), right: stringVals("b", "3")}, want: mergedStringVals("1", "a1", "2", "a2", "3", "b3")},
		// {name: "1 / 1 2 3", args: args{left: stringVals("a", "1"), right: stringVals("b", "1", "2", "3")}, want: mergedStringVals("1", "b1", "2", "b2", "3", "b3")},
		// {name: "2 / 1 2 3", args: args{left: stringVals("a", "2"), right: stringVals("b", "1", "2", "3")}, want: mergedStringVals("1", "b1", "2", "b2", "3", "b3")},
		// {name: "3 / 1 2 3", args: args{left: stringVals("a", "3"), right: stringVals("b", "1", "2", "3")}, want: mergedStringVals("1", "b1", "2", "b2", "3", "b3")},

		// {name: "nil / Values{nil}", args: args{left: nil, right: Attribs{nil}}, want: nil},
		// {name: "Values{nil} / nil", args: args{left: Attribs{nil}, right: nil}, want: nil},
		// {name: "Values{nil} / Values{nil, nil}", args: args{left: Attribs{nil}, right: Attribs{nil, nil}}, want: nil},
		// {name: "1 / 2 nil", args: args{left: stringVals("a", "1"), right: append(stringVals("b", "2"), nil)}, want: mergedStringVals("1", "a1", "2", "b2")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.args.left.CloneAndAppendNonExistingCloned(tt.args.right)
			if !intValsEqual(got, tt.want) {
				t.Errorf("MergeAttribs() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestAttribs_growPooled tests that growPooled only moves the attribs
// into a larger pooled slice when there is not enough spare capacity,
// and that the replaced slice is returned to the mempool.
func TestAttribs_growPooled(t *testing.T) {
	DrainAllMemPools()
	mempool.RegisterCallbacksWriterForTest(t, io.Discard)

	var nilAttribs Attribs
	grown := nilAttribs.growPooled(1)
	require.Empty(t, grown)
	require.GreaterOrEqual(t, cap(grown), 1)
	grown.Free()
	require.Zero(t, mempool.NumOutstanding(), "nil attribs are not put back")

	var a Attribs
	a.Add(NewString("a", "1"))
	require.Greater(t, cap(a), len(a))
	require.Same(t, &a[0], &a.growPooled(cap(a) - len(a))[0], "spare capacity is used in place")

	// Fill up the pooled slice so that it has to grow
	for i := len(a); i < cap(a); i++ {
		a.Add(NewInt("a"+strconv.Itoa(i), int64(i)))
	}
	attribs := a.Clone()
	grown = a.growPooled(2)
	require.Equal(t, attribs, grown, "grown slice has the same attribs")
	require.GreaterOrEqual(t, cap(grown), len(grown)+2)

	grown.Free()
	attribs.Free()
	require.Zero(t, mempool.NumOutstanding(), "every pooled attrib and slice was returned")
}

func TestAttribFromContext(t *testing.T) {
	_, ok := AttribFromContext[*Int](context.Background(), "invalid")
	require.False(t, ok, "attrib not added to context")

	ctx := ContextWithAttribs(context.Background(), NewInt("Int", 1))
	_, ok = AttribFromContext[*Int](ctx, "invalid")
	require.False(t, ok, "attrib not added to context")

	attrib, ok := AttribFromContext[*Int](ctx, "Int")
	require.True(t, ok, "attrib added to context")
	require.Equal(t, attrib, NewInt("Int", 1))
}
