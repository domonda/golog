package golog

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
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

// TestAttribs_AddToContext_Clones pins the non-consuming contract of
// Attribs.AddToContext: the returned context owns clones of the passed
// attribs and of the ones inherited from the parent context.
// A context must own its attribs because it has no end of life, so an
// attrib shared with another owner would be recycled by the mempool while
// the context still references it.
func TestAttribs_AddToContext_Clones(t *testing.T) {
	strA := NewString("a", "1")
	ctx1 := ContextWithAttribs(context.Background(), strA)
	require.NotSame(t, strA, AttribsFromContext(ctx1).Get("a"), "context cloned the passed attrib")
	require.Equal(t, strA, AttribsFromContext(ctx1).Get("a"), "clone has the same key and value")

	strB := NewString("b", "2")
	ctx2 := ContextWithAttribs(ctx1, strB)
	require.NotSame(t, strB, AttribsFromContext(ctx2).Get("b"), "context cloned the passed attrib")

	inheritedA := AttribsFromContext(ctx2).Get("a")
	require.NotSame(t, AttribsFromContext(ctx1).Get("a"), inheritedA, "attrib inherited from the parent context is a clone")
	require.Equal(t, strA, inheritedA, "clone has the same key and value")

	require.Len(t, AttribsFromContext(ctx1), 1, "parent context is unchanged")
	require.Equal(t, strA, AttribsFromContext(ctx1).Get("a"), "parent context is unchanged")
}

// TestAttribs_AddToContext_CallerKeepsOwnership tests that a caller can keep
// using and finally Free the Attribs it added to a context. Attribs.Add uses
// a pooled slice with spare capacity, so a context that appended into that
// slice instead of cloning would be corrupted by the next Add, and Free would
// nil out the attribs the context still references.
func TestAttribs_AddToContext_CallerKeepsOwnership(t *testing.T) {
	var attribs Attribs
	attribs.Add(NewString("a", "1"))
	require.Greater(t, cap(attribs), len(attribs), "pooled slice has spare capacity")

	parent := ContextWithAttribs(context.Background(), NewString("p", "parent"))
	ctx := attribs.AddToContext(parent)
	require.Equal(t, Attribs{NewString("a", "1"), NewString("p", "parent")}, AttribsFromContext(ctx))

	// Reusing the caller's slice must not overwrite the inherited "p" attrib
	attribs.Add(NewString("b", "2"))
	require.Equal(t, Attribs{NewString("a", "1"), NewString("p", "parent")}, AttribsFromContext(ctx))

	// Freeing the caller's attribs must not nil out the ones in the context
	attribs.Free()
	require.Equal(t, Attribs{NewString("a", "1"), NewString("p", "parent")}, AttribsFromContext(ctx))
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
