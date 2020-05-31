package golog

import (
	"context"
	"net/http"
)

// Value extends the Loggable interface and allows
// values to log themselfes and be referenced by a name.
type Value interface {
	Loggable

	// Name returns the name of the value
	Name() string
}

// Values is a Value slice with methods to manage and log them.
// Usually only one value with a given name is present in the slice,
// but this is not enforced.
// A slice is used instead of a map to preserve the order
// of values and to maximize allocation performance.
// Values implements the Loggable interface by logging
// the values in the slice in the given order.
type Values []Value

// Log implements the Loggable interface by logging
// the values in the slice in the given order.
func (v Values) Log(m *Message) {
	for _, value := range v {
		value.Log(m)
	}
}

// Get returns the first Value with the passed name or nil
func (v Values) Get(name string) Value {
	for _, value := range v {
		if value.Name() == name {
			return value
		}
	}
	return nil
}

// ReplaceOrAppend replaces the first value in the slice with the same name
// than the passed value, or else appends the passed value to the slice.
func (v *Values) ReplaceOrAppend(value Value) {
	name := value.Name()
	for i, existing := range *v {
		if existing.Name() == name {
			(*v)[i] = value
			return
		}
	}
	*v = append(*v, value)
}

var valuesContextKey struct{} // unique type for this package

// ValuesFromContext returns Values from the context
// or nil if the context has none.
func ValuesFromContext(ctx context.Context) Values {
	l, _ := ctx.Value(valuesContextKey).(Values)
	return l
}

// ValueFromContext returns a Value from the context
// or nil if the context has no value with the given name.
func ValueFromContext(ctx context.Context, name string) Value {
	return ValuesFromContext(ctx).Get(name)
}

// AddToContext returns a context with v added to it so it can
// be retrieved again with ValuesFromContext.
// If the context already has Values, then the result of
// MergeValues(ctxValues, v) will added to the context.
func (v Values) AddToContext(ctx context.Context) context.Context {
	if len(v) == 0 {
		return ctx
	}
	ctxValues := ValuesFromContext(ctx)
	merged := MergeValues(ctxValues, v)
	return context.WithValue(ctx, valuesContextKey, merged)
}

// AddToRequest returns a http.Request with v added to its context
// so it can be retrieved again with ValuesFromContext(request.Context()).
// If the context already has Values, then the result of
// MergeValues(ctxValues, v) will added to the context.
func (v Values) AddToRequest(request *http.Request) *http.Request {
	if len(v) == 0 {
		return request
	}
	ctx := v.AddToContext(request.Context())
	return request.WithContext(ctx)
}

// MergeValues merges a and b so that only one
// value with a given name is in the result.
// Order is preserved, values from a are removed if
// they are also present with the same name in b.
// Without name collisions, the result is identical to append(a, b).
// The result is a new slice, a and b are not changed.
func MergeValues(a, b Values) Values {
	if len(a) == 0 && len(b) == 0 {
		return nil
	}

	c := make(Values, len(a), len(a)+len(b))
	// Only copy values from a to c that don't exist with that name in b
	i := 0
	for _, aa := range a {
		name := aa.Name()
		nameInB := false
		for _, bb := range b {
			if name == bb.Name() {
				nameInB = true
				c = c[:len(c)-1]
				break
			}
		}
		if !nameInB {
			c[i] = aa
			i++
		}
	}
	// Then append uniquely name values from b
	return append(c, b...)
}
