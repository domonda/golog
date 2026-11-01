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
	*a = append(a.growPooled(1), attrib)
}

// growPooled returns the attribs of a in a slice
// with spare capacity for at least n more attribs.
//
// If a has not enough spare capacity, then its attribs are moved
// into a larger slice from the mempool and the slice of a is
// returned to the mempool, so the caller must be the exclusive
// owner of a and must not use it anymore.
func (a Attribs) growPooled(n int) Attribs {
	if len(a)+n <= cap(a) {
		return a
	}
	grown := attribsPool.GetOrMake(len(a), len(a)+n)
	copy(grown, a)
	if a != nil {
		attribsPool.ClearAndPutBack(a)
	}
	return grown
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
//
// The returned Attribs are owned by the context, so don't Add to them,
// don't Free them and don't pass them to a method that takes over
// ownership. A context has no end of life and never returns its attribs
// to the mempool, but Attribs.Add and Attribs.Free do, which would recycle
// attribs and the slice while the context still references them.
// Use Attribs.Clone to get a copy that the caller owns.
func AttribsFromContext(ctx context.Context) Attribs {
	attribs, _ := ctx.Value(&attribsCtxKey).(Attribs)
	return attribs
}

// AttribFromContext returns an attrib with a given key and type
// from a context or false for ok if no such attribute was
// added to the context.
//
// The returned attrib is owned by the context, so don't Free it,
// see AttribsFromContext.
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
// Use RequestWithOwnedAttribs for attribs that are built for the call.
func RequestWithAttribs(request *http.Request, attribs ...Attrib) *http.Request {
	return Attribs(attribs).AddToRequest(request)
}

// ContextWithOwnedAttribs returns a context that takes over ownership of the
// passed attribs instead of cloning them like ContextWithAttribs, overwriting
// any attribs with the same keys already added to the context.
//
// The caller must not use or Free the passed attribs anymore. Use it for
// attribs that are built for the call, like
//
//	ctx = golog.ContextWithOwnedAttribs(ctx, golog.NewUUID("requestID", requestID))
//
// where cloning allocates a second attrib and orphans the passed one,
// because a context never returns its attribs to the mempool and the caller
// has no reference left to Free them with.
// Attribs inherited from the parent context are still cloned, so that
// parent and child context stay independent.
//
// The added attribs can be retrieved from the context
// with AttribsFromContext.
func ContextWithOwnedAttribs(ctx context.Context, attribs ...Attrib) context.Context {
	return Attribs(attribs).addOwnedToContext(ctx)
}

// RequestWithOwnedAttribs returns an http.Request whose context takes over
// ownership of the passed attribs instead of cloning them like
// RequestWithAttribs, see ContextWithOwnedAttribs.
func RequestWithOwnedAttribs(request *http.Request, attribs ...Attrib) *http.Request {
	if len(attribs) == 0 {
		return request
	}
	return request.WithContext(Attribs(attribs).addOwnedToContext(request.Context()))
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

// addOwnedToContext returns a context that takes over ownership of the
// receiver's attribs instead of cloning them, see ContextWithOwnedAttribs.
// Only the attribs are taken over, not the slice holding them, which may
// be a variadic argument slice that never came from the mempool.
func (a Attribs) addOwnedToContext(ctx context.Context) context.Context {
	if len(a) == 0 {
		return ctx
	}
	parentAttribs := AttribsFromContext(ctx)
	owned := Attribs(attribsPool.GetOrMake(len(a), len(a)+a.numNonExisting(parentAttribs)))
	copy(owned, a) // Take over the attribs instead of cloning them
	// The parent context keeps ownership of its attribs, so clone those
	owned = owned.appendClonesOfNonExisting(a, parentAttribs)
	return context.WithValue(ctx, &attribsCtxKey, owned)
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

// numNonExisting returns the number of attribs from b
// with a key that is not present in a.
func (a Attribs) numNonExisting(b Attribs) int {
	num := 0
	for _, bAttrib := range b {
		if !a.Has(bAttrib.Key()) {
			num++
		}
	}
	return num
}

// appendClonesOfNonExisting appends clones of the attribs from b
// with a key that is not present in existing and returns the result.
//
// The receiver must have spare capacity for all appended clones, so that
// append can never allocate a new, non-pooled backing array that the
// mempool would hand out again, see Attribs.numNonExisting.
//
// Only existing is searched for the keys of b, never the appended clones,
// so attribs with a duplicate key within b are all appended. That keeps
// the merged attribs of a context and of a sub-logger identical, which
// both build on this method.
func (a Attribs) appendClonesOfNonExisting(existing, b Attribs) Attribs {
	for _, bAttrib := range b {
		if !existing.Has(bAttrib.Key()) {
			a = append(a, bAttrib.Clone())
		}
	}
	return a
}

// clonesOfNonExisting returns a pooled slice with clones of the attribs
// from b that have a key which is not present in a,
// or nil if there is no such attrib in b.
func (a Attribs) clonesOfNonExisting(b Attribs) Attribs {
	numNonExisting := a.numNonExisting(b)
	if numNonExisting == 0 {
		return nil
	}
	clones := Attribs(attribsPool.GetOrMake(0, numNonExisting))
	return clones.appendClonesOfNonExisting(a, b)
}

// appendNonExistingCloned appends clones of the attribs from b
// that are not already present in a identified by their key.
//
// The attribs of a are transferred to the returned slice,
// so a must not be used anymore, while b is not modified.
// The clones from b are appended to the backing array of a when it has
// spare capacity, else Attribs.growPooled moves them into a larger pooled
// slice and returns the slice of a to the mempool, so the caller must be
// the exclusive owner of a pooled a.
// Use Attribs.CloneAndAppendNonExistingCloned for attribs owned by someone else.
func (a Attribs) appendNonExistingCloned(b Attribs) Attribs {
	numNonExisting := a.numNonExisting(b)
	if numNonExisting == 0 {
		return a
	}
	// Make room for the clones upfront, growPooled may return
	// the slice of a to the mempool, so don't read a after it
	grown := a.growPooled(numNonExisting)
	// grown holds the attribs of a, so searching it for the keys of b
	// up to its current length is searching the attribs of a
	return grown.appendClonesOfNonExisting(grown, b)
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
	numNonExisting := a.numNonExisting(b)
	if numNonExisting == 0 {
		return a.Clone()
	}
	result := Attribs(attribsPool.GetOrMake(len(a), len(a)+numNonExisting))
	for i, aAttrib := range a {
		result[i] = aAttrib.Clone()
	}
	return result.appendClonesOfNonExisting(a, b)
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
