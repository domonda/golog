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

// AttribsFromContext returns the Attribs that were added
// to a context or nil.
func AttribsFromContext(ctx context.Context) Attribs {
	attribs, _ := ctx.Value(&attribsCtxKey).(Attribs)
	return attribs
}

// AddToContext returns a context with the Attribs
// added to it overwriting any attribs with the same keys
// already added to the context.
//
// The added attribs can be retreved from the context
// with AttribsFromContext.
func (a Attribs) AddToContext(ctx context.Context) context.Context {
	if len(a) == 0 {
		return ctx
	}
	mergedAttribs := a.AppendUnique(AttribsFromContext(ctx)...)
	return context.WithValue(ctx, &attribsCtxKey, mergedAttribs)
}

// AddToRequest returns a http.Request with the Attribs
// added to its context it overwriting any attribs with the same keys
// already added to the request context.
func (a Attribs) AddToRequest(request *http.Request) *http.Request {
	if len(a) == 0 {
		return request
	}
	ctx := a.AddToContext(request.Context())
	return request.WithContext(ctx)
}

// AppendUnique merges a and b so that keys are unique
// using attribs from a in case of identical keyed attribs in b.
//
// The slices left and right will never be modified,
// in case of a merge the result is always a new slice.
func (a Attribs) AppendUnique(b ...Attrib) Attribs {
	// Remove nil interfaces. They should not happen but robustness of logging is important!
	for i := len(a) - 1; i >= 0; i-- {
		if a[i] == nil {
			a = append(a[:i], a[i+1:]...)
		}
	}
	for i := len(b) - 1; i >= 0; i-- {
		if b[i] == nil {
			b = append(b[:i], b[i+1:]...)
		}
	}

	// No merge cases
	switch {
	case len(a) == 0:
		return b
	case len(b) == 0:
		return a
	}

	var result Attribs
	for _, bAttrib := range b {
		if a.Has(bAttrib.GetKey()) {
			// Ignore bAttrib value because the value from a is preferred
			continue
		}
		if result == nil {
			// Allocate new slice for merged result
			result = append(result, a...)
		}
		result = append(result, bAttrib)
	}
	if result == nil {
		// All keys of b were present in a
		// so result is identical to a
		return a
	}
	return result
}
