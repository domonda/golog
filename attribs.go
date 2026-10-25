package golog

import (
	"context"
	"net/http"

	"github.com/domonda/go-encjson"
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

var attribsCtxKey int

// Add appends an attribute to the Attribs slice.
// If a is nil or attrib is nil, the method is a no-op.
func (a *Attribs) Add(attrib Attrib) {
	if a == nil || attrib == nil {
		return // Be conservative and don't panic
	}
	if *a == nil {
		*a = attribsPool.GetOrMake(1, 1)
		(*a)[0] = attrib
		return
	}
	newLen := len(*a) + 1
	if newLen <= cap(*a) {
		*a = append(*a, attrib)
		return
	}
	added := attribsPool.GetOrMake(newLen, newLen)
	copy(added, *a)
	attribsPool.ClearAndPutBack(*a)
	added[newLen-1] = attrib
	*a = added
}

// Clone returns a new Attribs slice with the same attributes.
// The attributes are cloned by calling the Clone method of each attribute.
func (a Attribs) Clone() Attribs {
	if a == nil {
		return nil
	}
	c := attribsPool.GetOrMake(len(a), len(a))
	for i, attrib := range a {
		c[i] = attrib.Clone()
	}
	return c
}

// Free returns the Attribs to the pool.
func (a Attribs) Free() {
	if a == nil {
		return
	}
	for i := range a {
		a[i].Free()
	}
	attribsPool.ClearAndPutBack(a[:0])
}

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
		if attrib.Key() == key {
			return attrib
		}
	}
	return nil
}

// Has indicates if the Attribs contain an Attrib with the passed key
func (a Attribs) Has(key string) bool {
	for _, attrib := range a {
		if attrib.Key() == key {
			return true
		}
	}
	return false
}

// Len returns the length of the Attribs slice
func (a Attribs) Len() int {
	return len(a)
}

// AttribsFromContext returns the Attribs that were added
// to a context or nil.
func AttribsFromContext(ctx context.Context) Attribs {
	attribs, _ := ctx.Value(&attribsCtxKey).(Attribs)
	return attribs
}

// AttribFromContext returns an attrib with a given key and type
// from a context or false for ok if no such attribute was
// added to the context.
func AttribFromContext[T Attrib](ctx context.Context, key string) (attrib T, ok bool) {
	attrib, ok = AttribsFromContext(ctx).Get(key).(T)
	return attrib, ok
}

// ContextWithAttribs returns a context with the passed attribs
// added to it, overwriting any attribs with the same keys
// already added to the context.
//
// Clones of the passed attribs are added to the context,
// so the caller keeps ownership of them, see Attribs.AddToContext.
//
// The added attribs can be retrieved from the context
// with AttribsFromContext.
func ContextWithAttribs(ctx context.Context, attribs ...Attrib) context.Context {
	return Attribs(attribs).AddToContext(ctx)
}

// RequestWithAttribs returns an http.Request with the Attribs
// added to its context, overwriting any attribs with
// the same keys already added to the request context.
//
// Clones of the passed attribs are added to the request context,
// so the caller keeps ownership of them, see Attribs.AddToContext.
func RequestWithAttribs(request *http.Request, attribs ...Attrib) *http.Request {
	return Attribs(attribs).AddToRequest(request)
}

// AddToContext returns a context with the Attribs
// added to it, overwriting any attribs with the same keys
// already added to the context.
//
// The returned context gets clones of the receiver's attribs and of the
// attribs from the parent context that are not shadowed by a receiver attrib
// with the same key. The receiver and the parent context therefore keep
// ownership of their attribs and stay valid and usable independent of the
// returned context. A context must own its attribs because it has no end of
// life: an attrib that another owner may Free and return to the mempool
// (Attribs.Free, Logger.RemoveAttribs) would be recycled while the context
// still references it.
//
// The added attribs can be retrieved from the context
// with AttribsFromContext.
func (a Attribs) AddToContext(ctx context.Context) context.Context {
	if len(a) == 0 {
		return ctx
	}
	mergedAttribs := a.CloneAndAppendNonExistingCloned(AttribsFromContext(ctx))
	return context.WithValue(ctx, &attribsCtxKey, mergedAttribs)
}

// AddToRequest returns an http.Request with the Attribs
// added to its context, overwriting any attribs with
// the same keys already added to the request context.
//
// Clones of the receiver's attribs are added to the request context,
// so the receiver keeps ownership of them, see Attribs.AddToContext.
func (a Attribs) AddToRequest(request *http.Request) *http.Request {
	if len(a) == 0 {
		return request
	}
	ctx := a.AddToContext(request.Context())
	return request.WithContext(ctx)
}

// appendNonExistingCloned appends clones of the attribs from b
// that are not already present in a identified by their key.
//
// The attribs of a are transferred to the returned slice,
// so a must not be used anymore, while b is not modified.
// The clones from b are appended to the backing array of a when it has
// spare capacity, so the caller must be the exclusive owner of a.
// Use Attribs.CloneAndAppendNonExistingCloned for attribs owned by someone else.
func (a Attribs) appendNonExistingCloned(b Attribs) Attribs {
	if len(b) == 0 {
		return a
	}
	if cap(a) < len(a)+len(b) {
		// Transfer the attribs of a into a pooled slice with room for the
		// clones from b instead of growing a with append or Attribs.Add,
		// which could put a non-pooled slice into the mempool
		grown := attribsPool.GetOrMake(len(a), len(a)+len(b))
		copy(grown, a)
		a = grown
	}
	for _, bAttrib := range b {
		if !a.Has(bAttrib.Key()) {
			a = append(a, bAttrib.Clone())
		}
	}
	return a
}

// CloneAndAppendNonExistingCloned clones a and appends clones of
// attribs from b that are not already present in a
// identified by their key.
//
// The result is a new slice with cloned attribs and
// the slices a and b are not modified.
func (a Attribs) CloneAndAppendNonExistingCloned(b Attribs) Attribs {
	if len(a) == 0 {
		return b.Clone()
	}
	if len(b) == 0 {
		return a.Clone()
	}
	result := attribsPool.GetOrMake(len(a), len(a)+len(b))
	for i, aAttrib := range a {
		result[i] = aAttrib.Clone()
	}
	for _, bAttrib := range b {
		if !a.Has(bAttrib.Key()) {
			result = append(result, bAttrib.Clone())
		}
	}
	return result
}

// AppendJSON appends the attribs as a JSON object to the buffer.
// The JSON null value is appended for empty Attribs.
func (a Attribs) AppendJSON(buf []byte) []byte {
	if len(a) == 0 {
		return append(buf, "null"...)
	}
	buf = encjson.AppendObjectStart(buf)
	for _, attrib := range a {
		buf = attrib.AppendJSON(buf)
	}
	return encjson.AppendObjectEnd(buf)
}

// MarshalJSON implements encoding/json.Marshaler
// by returning a JSON object with the attribs as key-value pairs.
// The JSON null value is returned for empty Attribs.
func (a Attribs) MarshalJSON() ([]byte, error) {
	return a.AppendJSON(nil), nil
}
