package golog

import (
	"context"
	"net/http"
)

var (
	_ Loggable = Attribs(nil)
)

// Attribs is a Attrib slice with methods to manage and log them.
// Usually only one Attrib with a given key is present in the slice,
// but this is not enforced.
// A slice is used instead of a map to preserve the order
// of attributes and to maximize allocation performance.
//
// Attribs implements the Loggable interface by logging
// the attributes in the slice in the given order.
type Attribs []Attrib

// Log implements the Loggable interface by logging
// the attributes in the slice in the given order.
func (a Attribs) Log(m *Message) {
	for _, attrib := range a {
		attrib.Log(m)
	}
}

// Get returns the first Attrib with the passed key
// or nil if not Attrib was found.
func (a Attribs) Get(key string) Attrib {
	for _, attrib := range a {
		if attrib.GetKey() == key {
			return attrib
		}
	}
	return nil
}

// Has indicates if the Attribs contain an Attrib with the passed key
func (a Attribs) Has(key string) bool {
	for _, attrib := range a {
		if attrib.GetKey() == key {
			return true
		}
	}
	return false
}

// Len returns the length of the Attribs slice
func (a Attribs) Len() int {
	return len(a)
}

/*
// ReplaceOrAppend replaces the first value in the slice with the same key
// than the passed value, or else appends the passed value to the slice.
func (v *Attribs) ReplaceOrAppend(value Attrib) {
	name := value.Name()
	for i, existing := range *v {
		if existing.Name() == name {
			(*v)[i] = value
			return
		}
	}
	*v = append(*v, value)
}
*/

var attribsCtxKey int

// AttribsFromContext returns Attribs from the context
// or nil if the context has none.
func AttribsFromContext(ctx context.Context) Attribs {
	attribs, _ := ctx.Value(&attribsCtxKey).(Attribs)
	return attribs
}

// AttribFromContext returns an Attrib from the context
// or nil if the context has no Attrib with the given key.
func AttribFromContext(ctx context.Context, key string) Attrib {
	return AttribsFromContext(ctx).Get(key)
}

// AddAttribsToContext returns a context with the passed attribs added to it
// so they can be retrieved again with AttribsFromContext.
// If the context already has Attribs, then the result of
// MergeAttribs(ctxAttribs, attribs) will added to the context.
func AddAttribsToContext(ctx context.Context, attribs ...Attrib) context.Context {
	return Attribs(attribs).AddToContext(ctx)
}

// AddAttribsToRequest returns a http.Request with the passed attribs added to its context
// so they can be retrieved again with AttribsFromContext(request.Context()).
// If the context already has Attribs, then the result of
// MergeAttribs(ctxAttribs, attribs) will added to the context.
func AddAttribsToRequest(request *http.Request, attribs ...Attrib) *http.Request {
	return Attribs(attribs).AddToRequest(request)
}

// AddToContext returns a context with a added to it
// so it can be retrieved again with AttribsFromContext.
// If the context already has Attribs, then the result of
// MergeAttribs(ctxAttribs, a) will added to the context.
func (a Attribs) AddToContext(ctx context.Context) context.Context {
	if len(a) == 0 {
		return ctx
	}
	ctxAttribs := AttribsFromContext(ctx)
	mergedAttribs := MergeAttribs(ctxAttribs, a)
	return context.WithValue(ctx, &attribsCtxKey, mergedAttribs)
}

// AddToRequest returns a http.Request with v added to its context
// so it can be retrieved again with AttribsFromContext(request.Context()).
// If the context already has Attribs, then the result of
// MergeAttribs(ctxAttribs, a) will added to the context.
func (a Attribs) AddToRequest(request *http.Request) *http.Request {
	if len(a) == 0 {
		return request
	}
	ctx := a.AddToContext(request.Context())
	return request.WithContext(ctx)
}

// MergeAttribs merges left and right so that attribute keys are unique
// using attribs from right in case of identical keyed attribs in left
// (values from right overwrite values from left).
//
// The slices left and right will never be modified,
// in case of a merge the result is always a new slice.
func MergeAttribs(left, right Attribs) Attribs {
	// Remove nil interfaces. They should not happen but robustness of logging is important!
	for i := len(left) - 1; i >= 0; i-- {
		if left[i] == nil {
			left = append(left[:i], left[i+1:]...)
		}
	}
	for i := len(right) - 1; i >= 0; i-- {
		if right[i] == nil {
			right = append(right[:i], right[i+1:]...)
		}
	}

	// No merge cases
	switch {
	case len(left) == 0:
		return right
	case len(right) == 0:
		return left
	}

	merged := make(Attribs, len(left), len(left)+len(right))

	// Only copy attribs from left to merged that don't exist with that key in right
	i := 0
	for _, l := range left {
		key := l.GetKey()
		keyInRight := false
		for _, r := range right {
			if key == r.GetKey() {
				keyInRight = true
				merged = merged[:len(merged)-1]
				break
			}
		}
		if !keyInRight {
			merged[i] = l
			i++
		}
	}

	// Then append uniquely keyed attribs from right
	merged = append(merged, right...)

	return merged
}
