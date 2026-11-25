package golog

import (
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
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

// TestAttribs_mergeDuplicateKeys tests that the merging methods of Attribs
// agree about attribs with a duplicate key, which Attribs explicitly allows.
// Message.SubLoggerContext uses clonesOfNonExisting for the ctx attribs of
// the sub-logger and CloneAndAppendNonExistingCloned (via AddToContext) for
// the attribs of the returned context, so the two would carry a different set
// of attribs if one of them skipped a duplicate key that the other one appends.
func TestAttribs_mergeDuplicateKeys(t *testing.T) {
	a := Attribs{NewString("a", "1")}
	b := Attribs{
		NewString("k", "1"),
		NewString("k", "2"), // Duplicate key within b
		NewString("a", "2"), // Already present in a
	}
	expected := Attribs{NewString("a", "1"), NewString("k", "1"), NewString("k", "2")}

	require.Equal(t, expected, a.CloneAndAppendNonExistingCloned(b), "CloneAndAppendNonExistingCloned")
	require.Equal(t, expected[1:], a.clonesOfNonExisting(b), "clonesOfNonExisting")
}

// TestContextWithOwnedAttribs tests that the returned context takes over the
// passed attribs instead of cloning them, while attribs inherited from the
// parent context are still cloned so that parent and child context stay
// independent.
func TestContextWithOwnedAttribs(t *testing.T) {
	DrainAllMemPools()
	mempool.RegisterCallbacksWriterForTest(t, io.Discard)

	strA := NewString("a", "1")
	ctx1 := ContextWithOwnedAttribs(context.Background(), strA)
	require.Same(t, strA, AttribsFromContext(ctx1).Get("a"), "context took over the passed attrib")

	strB := NewString("b", "2")
	ctx2 := ContextWithOwnedAttribs(ctx1, strB)
	require.Same(t, strB, AttribsFromContext(ctx2).Get("b"), "context took over the passed attrib")

	inheritedA := AttribsFromContext(ctx2).Get("a")
	require.NotSame(t, strA, inheritedA, "attrib inherited from the parent context is a clone")
	require.Equal(t, strA, inheritedA, "clone has the same key and value")
	require.Equal(t, Attribs{strA}, AttribsFromContext(ctx1), "parent context is unchanged")

	// A passed attrib replaces the parent attrib with the same key,
	// like a nested request ID added by the HTTP middleware
	strA2 := NewString("a", "2")
	ctx3 := ContextWithOwnedAttribs(ctx2, strA2)
	require.Equal(t, Attribs{strA2, strB}, AttribsFromContext(ctx3), "passed attrib replaces the parent attrib with the same key")

	require.Equal(t, context.Background(), ContextWithOwnedAttribs(context.Background()), "no attribs returns the context unchanged")

	// Not cloning the passed attrib is the point of ContextWithOwnedAttribs:
	// ContextWithAttribs allocates a clone for the context and leaves the
	// passed attrib orphaned when it was built for the call, because a
	// context never returns its attribs to the mempool and the caller has
	// no reference left to free them with.
	before := mempool.NumOutstanding()
	ContextWithAttribs(context.Background(), NewString("c", "3"))
	numCloning := mempool.NumOutstanding() - before

	before = mempool.NumOutstanding()
	ContextWithOwnedAttribs(context.Background(), NewString("c", "3"))
	numOwned := mempool.NumOutstanding() - before

	require.Equal(t, numCloning-1, numOwned, "one pooled attrib less than ContextWithAttribs")
}

// TestRequestWithOwnedAttribs tests that the context of the returned request
// takes over the passed attribs, see TestContextWithOwnedAttribs.
func TestRequestWithOwnedAttribs(t *testing.T) {
	request := httptest.NewRequest("GET", "/", nil)
	str := NewString("a", "1")

	requestWithAttribs := RequestWithOwnedAttribs(request, str)
	require.Same(t, str, AttribsFromContext(requestWithAttribs.Context()).Get("a"), "request context took over the passed attrib")
	require.Nil(t, AttribsFromContext(request.Context()), "passed request is unchanged")
	require.Same(t, request, RequestWithOwnedAttribs(request), "no attribs returns the request unchanged")
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

// TestAttribs_Add tests that Add appends to a pooled slice and is a no-op
// for a nil receiver or a nil attrib, as documented, instead of panicking
// or storing a nil Attrib that would panic on the next Key call.
func TestAttribs_Add(t *testing.T) {
	var a Attribs
	a.Add(NewString("a", "1"))
	a.Add(nil)
	require.Equal(t, Attribs{NewString("a", "1")}, a, "nil attrib is not added")

	var nilAttribs *Attribs
	require.NotPanics(t, func() { nilAttribs.Add(NewString("b", "2")) }, "nil receiver is a no-op")

	a.Free()
}

// TestRequestWithAttribs tests that the context of the returned request
// gets clones of the passed attribs, so the caller keeps ownership of them,
// and that the passed request stays unchanged.
func TestRequestWithAttribs(t *testing.T) {
	request := httptest.NewRequest("GET", "/", nil)
	str := NewString("a", "1")

	requestWithAttribs := RequestWithAttribs(request, str)
	ctxAttrib := AttribsFromContext(requestWithAttribs.Context()).Get("a")
	require.NotSame(t, str, ctxAttrib, "request context has a clone of the passed attrib")
	require.Equal(t, str, ctxAttrib, "clone has the same key and value")
	require.Nil(t, AttribsFromContext(request.Context()), "passed request is unchanged")

	require.Same(t, request, RequestWithAttribs(request), "no attribs returns the request unchanged")
	require.Same(t, request, Attribs(nil).AddToRequest(request), "no attribs returns the request unchanged")

	// Attribs already in the request context are kept unless overwritten
	merged := RequestWithAttribs(requestWithAttribs, NewString("a", "2"), NewString("b", "3"))
	require.Equal(t, Attribs{NewString("a", "2"), NewString("b", "3")}, AttribsFromContext(merged.Context()))
}

// TestAttribs_MarshalJSON tests the JSON object written for Attribs,
// including the documented null for empty Attribs.
func TestAttribs_MarshalJSON(t *testing.T) {
	for _, tc := range []struct {
		name    string
		attribs Attribs
		want    string
	}{
		{name: "nil", attribs: nil, want: `null`},
		{name: "empty", attribs: Attribs{}, want: `null`},
		{name: "one", attribs: Attribs{NewString("a", "1")}, want: `{"a":"1"}`},
		{
			name:    "several types",
			attribs: Attribs{NewString("str", "x"), NewInt("int", -1), NewBool("bool", true), NewNil("nil")},
			want:    `{"str":"x","int":-1,"bool":true,"nil":null}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data, err := tc.attribs.MarshalJSON()
			require.NoError(t, err)
			require.JSONEq(t, tc.want, string(data))
			require.True(t, json.Valid(data), "valid JSON: %s", data)

			// AppendJSON appends to the passed buffer as the next JSON value,
			// separated by a comma from a preceding value
			require.Equal(t, "["+string(data), string(tc.attribs.AppendJSON([]byte("["))))
			require.Equal(t, "[1,"+string(data), string(tc.attribs.AppendJSON([]byte("[1"))))

			// encoding/json uses MarshalJSON
			wrapped, err := json.Marshal(struct{ Attribs Attribs }{tc.attribs})
			require.NoError(t, err)
			require.JSONEq(t, `{"Attribs":`+tc.want+`}`, string(wrapped))
		})
	}
}

// TestGoroutineID tests that goroutineID parses the ID of the calling
// goroutine from its stack trace, which Attribs.MarshalJSON uses to find
// Attribs that reference themselves, so a new Go version that changes
// the format of the stack trace fails this test.
func TestGoroutineID(t *testing.T) {
	id, ok := goroutineID()
	require.True(t, ok, "goroutine ID parsed")
	require.NotZero(t, id)
	ids := make(chan uint64)
	go func() {
		other, _ := goroutineID()
		ids <- other
	}()
	require.NotEqual(t, id, <-ids, "IDs of different goroutines")
}

// TestAttribs_MarshalJSON_LoggableFuncCycle tests a cycle through
// Attribs.MarshalJSON that no walk of the value could find, because
// a LoggableFunc logs the value that holds it.
func TestAttribs_MarshalJSON_LoggableFuncCycle(t *testing.T) {
	holder := &testAttribsHolder{}
	holder.A = Attribs{NewAny("f", LoggableFunc(func(m *Message) { m.Any("h", holder) }))}
	requireReturns(t, func() {
		// The inner marshalling of holder writes the cycle error as string
		j, err := json.Marshal(holder)
		require.NoError(t, err)
		require.Contains(t, string(j), "golog: encountered a cycle via golog.Attribs")
	})
}
