package golog

import (
	"reflect"
	"strings"
)

// Interface types for the IsNull and IsZero method checks used by the
// omitnull and omitzero modifiers. Declared once at package level so we
// can cheaply test whether reflect.PointerTo(T) implements them.
var (
	isNullIfaceType = reflect.TypeOf((*interface{ IsNull() bool })(nil)).Elem()
	isZeroIfaceType = reflect.TypeOf((*interface{ IsZero() bool })(nil)).Elem()
)

// structFieldDirectives is the parsed result of a struct tag value such as
// "name,omitempty,redact" — see parseStructFieldDirectives.
type structFieldDirectives struct {
	key       string // explicit tag name; empty means "use field.Name"
	skip      bool   // tag value was exactly "-"
	redact    bool
	omitempty bool
	omitzero  bool
	omitnull  bool
}

// parseStructFieldDirectives parses a raw struct tag value (the string after
// the tag name, e.g. the "name,omitempty,redact" part of `json:"name,omitempty,redact"`)
// into a structFieldDirectives. The caller is responsible for looking the tag
// up on the struct field; this helper only parses the value.
//
// Rules:
//   - Bare "-" (no comma) → skip the field.
//   - Empty name ("" or ",...") → key left empty; caller substitutes field.Name.
//   - Recognized modifiers: omitempty, omitzero, omitnull, redact (also spelled
//     "redacted" for the redact modifier).
//   - Unknown modifier tokens are ignored silently (matches encoding/json).
//   - Whitespace around the name and each modifier is trimmed.
func parseStructFieldDirectives(value string) structFieldDirectives {
	var d structFieldDirectives

	name, rest, hasRest := strings.Cut(value, ",")
	name = strings.TrimSpace(name)

	if name == "-" && !hasRest {
		d.skip = true
		return d
	}
	d.key = name

	for rest != "" {
		var mod string
		mod, rest, _ = strings.Cut(rest, ",")
		switch strings.TrimSpace(mod) {
		case "redact", "redacted":
			d.redact = true
		case "omitempty":
			d.omitempty = true
		case "omitzero":
			d.omitzero = true
		case "omitnull":
			d.omitnull = true
		}
	}
	return d
}

// shouldOmitStructField reports whether fv should be suppressed from log
// output per the given directives. The three suppression modifiers are OR'd:
// any passing check suppresses the field.
func shouldOmitStructField(fv reflect.Value, d structFieldDirectives) bool {
	if d.omitnull {
		// A nil pointer or interface is definitively null. Short-circuit
		// before the interface assertion: a nil *T with a value-receiver
		// IsNull method would otherwise panic when IsNull auto-dereferences
		// the nil pointer.
		if (fv.Kind() == reflect.Pointer || fv.Kind() == reflect.Interface) && fv.IsNil() {
			return true
		}
		if isNull, ok := invokeIsNull(fv); ok {
			if isNull {
				return true
			}
		} else if isZeroValueForOmitzero(fv) {
			return true
		}
	}
	if d.omitzero && isZeroValueForOmitzero(fv) {
		return true
	}
	if d.omitempty && isEmptyValueForOmitempty(fv) {
		return true
	}
	return false
}

// invokeIsNull calls IsNull() bool on fv if its dynamic type implements the
// interface, handling both value-receiver and pointer-receiver methods.
// Returns (result, true) if the method was called, or (false, false) if the
// type does not implement IsNull at all.
//
// The pointer-receiver path needs an addressable form of fv. If fv is
// already addressable (because its enclosing struct was reached via a
// pointer), we take its address directly. Otherwise we allocate a fresh
// copy so we can call the method without panicking.
func invokeIsNull(fv reflect.Value) (result, ok bool) {
	// Value-receiver fast path: no allocation, works when fv's type itself
	// is in the method set.
	if n, ok := fv.Interface().(interface{ IsNull() bool }); ok {
		return n.IsNull(), true
	}
	// Pointer-receiver path: only applies if *T has the method.
	if !reflect.PointerTo(fv.Type()).Implements(isNullIfaceType) {
		return false, false
	}
	addr := asAddressableValue(fv).Addr()
	return addr.Interface().(interface{ IsNull() bool }).IsNull(), true
}

// invokeIsZero calls IsZero() bool on fv if its dynamic type implements the
// interface, handling both value-receiver and pointer-receiver methods.
// See invokeIsNull for the addressability strategy.
func invokeIsZero(fv reflect.Value) (result, ok bool) {
	if z, ok := fv.Interface().(interface{ IsZero() bool }); ok {
		return z.IsZero(), true
	}
	if !reflect.PointerTo(fv.Type()).Implements(isZeroIfaceType) {
		return false, false
	}
	addr := asAddressableValue(fv).Addr()
	return addr.Interface().(interface{ IsZero() bool }).IsZero(), true
}

// asAddressableValue returns fv if it is already addressable, otherwise a
// fresh addressable copy of fv. The returned value has the same type as
// fv; call .Addr() on it to obtain a pointer.
//
// Struct fields obtained by walking a value (reflect.ValueOf(structValue))
// are not addressable; fields reached via a pointer
// (reflect.ValueOf(&structValue).Elem()) are. The copy path costs one
// allocation and is only used when fv is not already addressable AND its
// pointer type implements the method we want to call.
func asAddressableValue(fv reflect.Value) reflect.Value {
	if fv.CanAddr() {
		return fv
	}
	tmp := reflect.New(fv.Type()).Elem()
	tmp.Set(fv)
	return tmp
}

// isZeroValueForOmitzero reports whether fv should be treated as zero for
// the omitzero modifier. It implements a superset of encoding/json's Go 1.24
// omitzero rule: if the value's type has an IsZero() bool method, that method
// is called first — and if it returns false, we ALSO check the Go zero value
// via reflect.Value.IsZero. Either one returning true is enough to suppress.
//
// The union matters for types whose IsZero method does not inspect every
// field (or inspects logical nullness rather than bit-for-bit zero): we still
// suppress a bit-for-bit uninitialized default. For types without an IsZero
// method, this reduces to plain reflect.Value.IsZero, which is the Go spec's
// "zero value" definition used by encoding/json's omitzero fallback.
//
// Both value-receiver and pointer-receiver IsZero methods are honored;
// see invokeIsZero for the addressability strategy.
func isZeroValueForOmitzero(fv reflect.Value) bool {
	// Nil pointer or interface is zero by definition. Short-circuit before
	// the interface assertion so a nil *T with a value-receiver IsZero
	// method does not panic on nil-pointer dereference.
	if (fv.Kind() == reflect.Pointer || fv.Kind() == reflect.Interface) && fv.IsNil() {
		return true
	}
	if isZero, ok := invokeIsZero(fv); ok && isZero {
		return true
	}
	return fv.IsZero()
}

// isEmptyValueForOmitempty mirrors encoding/json's omitempty definition:
// false, 0, a nil pointer or interface, or an empty array/slice/map/string.
// It does NOT treat zero structs or time.Time{} as empty — that is what
// omitzero is for.
func isEmptyValueForOmitempty(fv reflect.Value) bool {
	switch fv.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return fv.Len() == 0
	case reflect.Bool:
		return !fv.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fv.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return fv.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return fv.Float() == 0
	case reflect.Interface, reflect.Pointer:
		return fv.IsNil()
	}
	return false
}
