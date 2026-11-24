package golog

import (
	"bytes"
	"encoding"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/domonda/go-encjson"
)

// Attrib extends the Loggable interface and allows
// attributes to log themselves and be referenced by a key.
type Attrib interface {
	Loggable
	fmt.Stringer

	// Key returns the attribute key
	Key() string

	// Value returns the attribute value
	Value() any

	// ValueString returns the attribute value
	// formatted as string
	ValueString() string

	// AppendJSON appends the attribute key and value
	// to the buffer in JSON format
	AppendJSON(buf []byte) []byte

	// Clone returns a copy of the attribute
	Clone() Attrib

	// Free returns the attribute to the pool
	Free()
}

// SliceAttrib extends Attrib for attributes that contain multiple values.
type SliceAttrib interface {
	Attrib

	// Len returns the number of values in the slice.
	Len() int
}

// Attrib implementations
var (
	_ Attrib      = &Nil{}
	_ Attrib      = &Any{}
	_ Attrib      = &Bool{}
	_ SliceAttrib = &Bools{}
	_ Attrib      = &Int{}
	_ SliceAttrib = &Ints{}
	_ Attrib      = &Uint{}
	_ SliceAttrib = &Uints{}
	_ Attrib      = &Float{}
	_ SliceAttrib = &Floats{}
	_ Attrib      = &String{}
	_ SliceAttrib = &Strings{}
	_ Attrib      = &Error{}
	_ SliceAttrib = &Errors{}
	_ Attrib      = &UUID{}
	_ SliceAttrib = &UUIDs{}
	_ Attrib      = &JSON{}
)

// Nil

type Nil struct {
	key string
}

func NewNil(key string) *Nil {
	a := nilPool.GetOrNew()
	a.key = key
	return a
}

func (a *Nil) Clone() Attrib {
	return NewNil(a.key)
}

func (a *Nil) Free() {
	nilPool.ZeroAndPutBack(a)
}

func (a *Nil) Key() string         { return a.key }
func (a *Nil) Value() any          { return nil }
func (a *Nil) ValueString() string { return "<nil>" }

func (a *Nil) Log(m *Message) {
	m.Nil(a.key)
}

func (a *Nil) AppendJSON(buf []byte) []byte {
	return encjson.AppendNull(encjson.AppendKey(buf, a.key))
}

func (a *Nil) String() string {
	return fmt.Sprintf("Nil{%s}", a.key)
}

// Any

type Any struct {
	key string
	val any
}

func NewAny(key string, val any) *Any {
	a := anyPool.GetOrNew()
	a.key = key
	a.val = val
	return a
}

func (a *Any) Clone() Attrib {
	return NewAny(a.key, a.val)
}

func (a *Any) Free() {
	anyPool.ZeroAndPutBack(a)
}

func (a *Any) Key() string { return a.key }
func (a *Any) Value() any  { return a.val }
func (a *Any) ValueString() string {
	if a.val != nil {
		if b, ok := byteSliceOf(reflect.ValueOf(a.val)); ok {
			// Like Message.StrBytes and Message.Any
			if b == nil || isNullValue(reflect.ValueOf(a.val)) {
				return "<nil>"
			}
			return bytesAsString(b)
		}
	}
	if _, ok := a.val.([]any); ok || hasOwnByteSliceFormat(reflect.ValueOf(a.val)) {
		// Like they are logged instead of Go syntax of their types,
		// like bytes for json.RawMessage or a net.IP, which could
		// also be a secret that the type redacts when it is logged
		return string(appendAnyJSON(nil, a.val))
	}
	return fmt.Sprintf("%#v", a.val)
}

func (a *Any) Log(m *Message) {
	m.Any(a.key, a.val)
}

func (a Any) AppendJSON(buf []byte) []byte {
	return appendAnyJSON(encjson.AppendKey(buf, a.key), a.val)
}

// appendAnyJSON appends val as JSON value to buf with a comma separator
// after a preceding value, written like Message.Any writes it,
// also for the elements of a []any.
func appendAnyJSON(buf []byte, val any) []byte {
	switch v := val.(type) {
	case nil:
		return encjson.AppendNull(buf)
	case bool:
		return encjson.AppendBool(buf, v)
	case int:
		return encjson.AppendInt(buf, int64(v))
	case int8:
		return encjson.AppendInt(buf, int64(v))
	case int16:
		return encjson.AppendInt(buf, int64(v))
	case int32:
		return encjson.AppendInt(buf, int64(v))
	case int64:
		return encjson.AppendInt(buf, v)
	case uint:
		return encjson.AppendUint(buf, uint64(v))
	case uint8:
		return encjson.AppendUint(buf, uint64(v))
	case uint16:
		return encjson.AppendUint(buf, uint64(v))
	case uint32:
		return encjson.AppendUint(buf, uint64(v))
	case uint64:
		return encjson.AppendUint(buf, v)
	case float32:
		return encjson.AppendFloat(buf, float64(v))
	case float64:
		return encjson.AppendFloat(buf, v)
	case string:
		return encjson.AppendString(buf, v)
	case []byte:
		return appendBytesJSON(buf, v)
	case time.Time:
		if v.IsZero() {
			// Like Message.Time and Message.Any
			return encjson.AppendNull(buf)
		}
		return encjson.AppendTime(buf, v, DefaultTimeFormat)
	case [16]byte:
		if IsNilUUID(v) {
			// Like Message.UUID and Message.Any
			return encjson.AppendNull(buf)
		}
		return encjson.AppendUUID(buf, v)
	case json.RawMessage:
		if len(v) == 0 {
			// An empty value is no valid JSON
			return encjson.AppendNull(buf)
		}
		// Compact like Message.JSON, because a line break
		// in pretty printed JSON would split the log line.
		// Compact appends nothing to compacted in case of an error.
		compacted := bytes.NewBuffer(appendJSONSeparator(buf))
		if json.Compact(compacted, v) != nil {
			// Like Message.JSON logs invalid JSON as error string,
			// appending it would make the whole JSON invalid
			return encjson.AppendString(buf, string(v))
		}
		return compacted.Bytes()
	case Loggable:
		// Like Message.Any, as JSON object of the attribs that it logs
		// instead of encoding/json, which can't see the unexported
		// fields of attribs and would encode Attribs as [{},{}]
		if rv := reflect.ValueOf(v); rv.Kind() == reflect.Pointer && rv.IsNil() {
			return encjson.AppendNull(buf)
		}
		return appendLoggableJSON(buf, nil, v)
	case error:
		// Like Message.Any, the error message instead of encoding/json,
		// which encodes most errors as empty object
		if rv := reflect.ValueOf(v); rv.Kind() == reflect.Pointer && rv.IsNil() {
			return encjson.AppendNull(buf)
		}
		return encjson.AppendString(buf, v.Error())
	case []any:
		// Every element written like a single value, for example
		// the mixed elements of a slice passed to a MessageCallback.
		// Not encjson.AppendArrayStart, which would prepend a comma
		// to a []any that is the first element of a []any
		buf = append(appendJSONSeparator(buf), '[')
		for _, elem := range v {
			buf = appendAnyJSON(buf, elem)
		}
		return encjson.AppendArrayEnd(buf)
	default:
		rv := reflect.ValueOf(v)
		// A pointer to a byte slice type with a format of its own,
		// also through several pointers: follow the pointers only up to
		// the first type that has the format methods, which can have
		// pointer receivers, and write it with the cases above
		if rv.Kind() == reflect.Pointer && hasOwnByteSliceFormat(rv) && !implementsOwnFormat(rv.Type()) {
			for rv.Kind() == reflect.Pointer && !implementsOwnFormat(rv.Type()) {
				if rv.IsNil() {
					return encjson.AppendNull(buf)
				}
				rv = rv.Elem()
			}
			return appendAnyJSON(buf, rv.Interface())
		}
		// Named byte slice types and pointers to byte slices
		if b, ok := byteSliceOf(rv); ok {
			if isNullValue(rv) {
				// Like Message.Any
				return encjson.AppendNull(buf)
			}
			return appendBytesJSON(buf, b)
		}
		// A byte slice type with a format of its own, like net.IP,
		// only with String instead of as base64 by encoding/json,
		// which uses MarshalJSON and MarshalText like Message.Any
		if s, ok := v.(fmt.Stringer); ok && hasOwnByteSliceFormat(rv) {
			_, marshaler := v.(json.Marshaler)
			_, textMarshaler := v.(encoding.TextMarshaler)
			if !marshaler && !textMarshaler {
				if rv.Kind() == reflect.Pointer && rv.IsNil() {
					// Don't call a String method on a nil pointer
					return encjson.AppendNull(buf)
				}
				return encjson.AppendString(buf, s.String())
			}
		}
		// Slower escape hatch for all other types
		j, err := json.Marshal(v)
		if err != nil {
			return encjson.AppendString(buf, err.Error())
		}
		return encjson.AppendJSON(buf, j)
	}
}

// appendJSONSeparator appends a comma to buf if buf ends
// with a JSON value, like the encjson functions do
// before they append a value.
func appendJSONSeparator(buf []byte) []byte {
	if l := len(buf); l > 0 && buf[l-1] != ':' && buf[l-1] != '[' {
		return append(buf, ',')
	}
	return buf
}

// appendBytesJSON appends the bytes like Message.StrBytes logs them:
// as string if they are valid UTF-8, else encoded with BinaryEncoder,
// or null for nil.
func appendBytesJSON(buf, b []byte) []byte {
	switch {
	case b == nil:
		return encjson.AppendNull(buf)
	case utf8.Valid(b):
		// Valid UTF-8 needs no conversion to a string
		return encjson.AppendStringBytes(buf, b)
	default:
		return encjson.AppendString(buf, BinaryEncoder(b))
	}
}

func (a *Any) String() string {
	return fmt.Sprintf("Any{%q: %s}", a.key, a.ValueString())
}

// Bool

type Bool struct {
	key string
	val bool
}

func NewBool(key string, val bool) *Bool {
	a := boolPool.GetOrNew()
	a.key = key
	a.val = val
	return a
}

func (a *Bool) Clone() Attrib {
	return NewBool(a.key, a.val)
}

func (a *Bool) Free() {
	boolPool.ZeroAndPutBack(a)
}

func (a Bool) Key() string         { return a.key }
func (a Bool) Value() any          { return a.val }
func (a Bool) ValueString() string { return fmt.Sprintf("%#v", a.val) }
func (a *Bool) ValueBool() bool    { return a.val }

func (a *Bool) Log(m *Message) {
	m.Bool(a.key, a.val)
}

func (a *Bool) AppendJSON(buf []byte) []byte {
	return encjson.AppendBool(encjson.AppendKey(buf, a.key), a.val)
}

func (a *Bool) String() string {
	return fmt.Sprintf("Bool{%q: %s}", a.key, a.ValueString())
}

type Bools struct {
	key  string
	vals []bool
}

func NewBools(key string, vals []bool) *Bools {
	a := boolsPool.GetOrNew()
	a.key = key
	a.vals = vals
	return a
}

func NewBoolsCopy(key string, vals []bool) *Bools {
	return NewBools(key, slices.Clone(vals))
}

func (a *Bools) Clone() Attrib {
	return NewBools(a.key, a.vals)
}

func (a *Bools) Free() {
	boolsPool.ZeroAndPutBack(a)
}

func (a *Bools) Key() string         { return a.key }
func (a *Bools) Value() any          { return a.vals }
func (a *Bools) ValueString() string { return fmt.Sprintf("%#v", a.vals) }
func (a *Bools) ValueBools() []bool  { return a.vals }

func (a *Bools) Log(m *Message) {
	m.Bools(a.key, a.vals)
}

func (a *Bools) AppendJSON(buf []byte) []byte {
	buf = encjson.AppendArrayStart(encjson.AppendKey(buf, a.key))
	for _, val := range a.vals {
		buf = encjson.AppendBool(buf, val)
	}
	return encjson.AppendArrayEnd(buf)
}

func (a *Bools) String() string {
	return fmt.Sprintf("Bools{%q: %s}", a.key, a.ValueString())
}

func (a *Bools) Len() int { return len(a.vals) }

// Int

type Int struct {
	key string
	val int64
}

func NewInt(key string, val int64) *Int {
	a := intPool.GetOrNew()
	a.key = key
	a.val = val
	return a
}

func (a *Int) Clone() Attrib {
	return NewInt(a.key, a.val)
}

func (a *Int) Free() {
	intPool.ZeroAndPutBack(a)
}

func (a *Int) Key() string         { return a.key }
func (a *Int) Value() any          { return a.val }
func (a *Int) ValueString() string { return fmt.Sprintf("%#v", a.val) }
func (a *Int) ValueInt() int64     { return a.val }

func (a *Int) Log(m *Message) {
	m.Int64(a.key, a.val)
}

func (a *Int) AppendJSON(buf []byte) []byte {
	return encjson.AppendInt(encjson.AppendKey(buf, a.key), a.val)
}

func (a *Int) String() string {
	return fmt.Sprintf("Int{%q: %s}", a.key, a.ValueString())
}

type Ints struct {
	key  string
	vals []int64
}

func NewInts(key string, vals []int64) *Ints {
	a := intsPool.GetOrNew()
	a.key = key
	a.vals = vals
	return a
}

func NewIntsCopy[T ~int | ~int8 | ~int16 | ~int32 | ~int64](key string, vals []T) *Ints {
	if vals == nil {
		return NewInts(key, nil)
	}
	ints := make([]int64, len(vals))
	for i, val := range vals {
		ints[i] = int64(val)
	}
	return NewInts(key, ints)
}

func (a *Ints) Clone() Attrib {
	return NewInts(a.key, a.vals)
}

func (a *Ints) Free() {
	intsPool.ZeroAndPutBack(a)
}

func (a *Ints) Key() string         { return a.key }
func (a *Ints) Value() any          { return a.vals }
func (a *Ints) ValueString() string { return fmt.Sprintf("%#v", a.vals) }
func (a *Ints) ValueInts() []int64  { return a.vals }

func (a *Ints) Log(m *Message) {
	m.Int64s(a.key, a.vals)
}

func (a *Ints) AppendJSON(buf []byte) []byte {
	buf = encjson.AppendArrayStart(encjson.AppendKey(buf, a.key))
	for _, val := range a.vals {
		buf = encjson.AppendInt(buf, val)
	}
	return encjson.AppendArrayEnd(buf)
}

func (a *Ints) String() string {
	return fmt.Sprintf("Ints{%q: %s}", a.key, a.ValueString())
}

func (a *Ints) Len() int { return len(a.vals) }

// Uint

type Uint struct {
	key string
	val uint64
}

func NewUint(key string, val uint64) *Uint {
	a := uintPool.GetOrNew()
	a.key = key
	a.val = val
	return a
}

func (a *Uint) Clone() Attrib {
	return NewUint(a.key, a.val)
}

func (a *Uint) Free() {
	uintPool.ZeroAndPutBack(a)
}

func (a *Uint) Key() string         { return a.key }
func (a *Uint) Value() any          { return a.val }
func (a *Uint) ValueString() string { return fmt.Sprintf("%#v", a.val) }
func (a *Uint) ValueUint() uint64   { return a.val }

func (a *Uint) Log(m *Message) {
	m.Uint64(a.key, a.val)
}

func (a *Uint) AppendJSON(buf []byte) []byte {
	return encjson.AppendUint(encjson.AppendKey(buf, a.key), a.val)
}

func (a *Uint) String() string {
	return fmt.Sprintf("Uint{%q: %s}", a.key, a.ValueString())
}

type Uints struct {
	key  string
	vals []uint64
}

func NewUints(key string, vals []uint64) *Uints {
	a := uintsPool.GetOrNew()
	a.key = key
	a.vals = vals
	return a
}

func NewUintsCopy[T ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64](key string, vals []T) *Uints {
	if vals == nil {
		return NewUints(key, nil)
	}
	uints := make([]uint64, len(vals))
	for i, val := range vals {
		uints[i] = uint64(val)
	}
	return NewUints(key, uints)
}

func (a *Uints) Clone() Attrib {
	return NewUints(a.key, a.vals)
}

func (a *Uints) Free() {
	uintsPool.ZeroAndPutBack(a)
}

func (a *Uints) Key() string          { return a.key }
func (a *Uints) Value() any           { return a.vals }
func (a *Uints) ValueString() string  { return fmt.Sprintf("%#v", a.vals) }
func (a *Uints) ValueUints() []uint64 { return a.vals }

func (a *Uints) Log(m *Message) {
	m.Uint64s(a.key, a.vals)
}

func (a *Uints) AppendJSON(buf []byte) []byte {
	buf = encjson.AppendArrayStart(encjson.AppendKey(buf, a.key))
	for _, val := range a.vals {
		buf = encjson.AppendUint(buf, val)
	}
	return encjson.AppendArrayEnd(buf)
}

func (a *Uints) String() string {
	return fmt.Sprintf("Uints{%q: %s}", a.key, a.ValueString())
}

func (a *Uints) Len() int { return len(a.vals) }

// Float

type Float struct {
	key string
	val float64
}

func NewFloat(key string, val float64) *Float {
	a := floatPool.GetOrNew()
	a.key = key
	a.val = val
	return a
}

func (a *Float) Clone() Attrib {
	return NewFloat(a.key, a.val)
}

func (a *Float) Free() {
	floatPool.ZeroAndPutBack(a)
}

func (a *Float) Key() string         { return a.key }
func (a *Float) Value() any          { return a.val }
func (a *Float) ValueString() string { return fmt.Sprintf("%#v", a.val) }
func (a *Float) ValueFloat() float64 { return a.val }

func (a *Float) Log(m *Message) {
	m.Float(a.key, a.val)
}

func (a *Float) AppendJSON(buf []byte) []byte {
	return encjson.AppendFloat(encjson.AppendKey(buf, a.key), a.val)
}

func (a *Float) String() string {
	return fmt.Sprintf("Float{%q: %s}", a.key, a.ValueString())
}

type Floats struct {
	key  string
	vals []float64
}

func NewFloats(key string, vals []float64) *Floats {
	a := floatsPool.GetOrNew()
	a.key = key
	a.vals = vals
	return a
}

func NewFloatsCopy[T ~float32 | ~float64](key string, vals []T) *Floats {
	if vals == nil {
		return NewFloats(key, nil)
	}
	floats := make([]float64, len(vals))
	for i, val := range vals {
		floats[i] = float64(val)
	}
	return NewFloats(key, floats)
}

func (a *Floats) Clone() Attrib {
	return NewFloats(a.key, a.vals)
}

func (a *Floats) Free() {
	floatsPool.ZeroAndPutBack(a)
}

func (a *Floats) Key() string            { return a.key }
func (a *Floats) Value() any             { return a.vals }
func (a *Floats) ValueString() string    { return fmt.Sprintf("%#v", a.vals) }
func (a *Floats) ValueFloats() []float64 { return a.vals }

func (a *Floats) Log(m *Message) {
	m.Floats(a.key, a.vals)
}

func (a *Floats) AppendJSON(buf []byte) []byte {
	buf = encjson.AppendArrayStart(encjson.AppendKey(buf, a.key))
	for _, val := range a.vals {
		buf = encjson.AppendFloat(buf, val)
	}
	return encjson.AppendArrayEnd(buf)
}

func (a *Floats) String() string {
	return fmt.Sprintf("Floats{%q: %s}", a.key, a.ValueString())
}

func (a *Floats) Len() int { return len(a.vals) }

// String

type String struct {
	key string
	val string
}

func NewString(key string, val string) *String {
	a := stringPool.GetOrNew()
	a.key = key
	a.val = val
	return a
}

func (a *String) Clone() Attrib {
	return NewString(a.key, a.val)
}

func (a *String) Free() {
	stringPool.ZeroAndPutBack(a)
}

func (a *String) Key() string         { return a.key }
func (a *String) Value() any          { return a.val }
func (a *String) ValueString() string { return a.val }

func (a *String) Log(m *Message) {
	m.Str(a.key, a.val)
}

func (a *String) AppendJSON(buf []byte) []byte {
	return encjson.AppendString(encjson.AppendKey(buf, a.key), a.val)
}

func (a *String) String() string {
	return fmt.Sprintf("String{%q: %q}", a.key, a.val)
}

type Strings struct {
	key  string
	vals []string
}

func NewStrings(key string, vals []string) *Strings {
	a := stringsPool.GetOrNew()
	a.key = key
	a.vals = vals
	return a
}

func NewStringsCopy[T ~string](key string, vals []T) *Strings {
	if vals == nil {
		return NewStrings(key, nil)
	}
	strings := make([]string, len(vals))
	for i, val := range vals {
		strings[i] = string(val)
	}
	return NewStrings(key, strings)
}

func (a *Strings) Clone() Attrib {
	return NewStrings(a.key, a.vals)
}

func (a *Strings) Free() {
	stringsPool.ZeroAndPutBack(a)
}

func (a *Strings) Key() string            { return a.key }
func (a *Strings) Value() any             { return a.vals }
func (a *Strings) ValueString() string    { return fmt.Sprintf("%#v", a.vals) }
func (a *Strings) ValueStrings() []string { return a.vals }

func (a *Strings) Log(m *Message) {
	m.Strs(a.key, a.vals)
}

func (a *Strings) AppendJSON(buf []byte) []byte {
	buf = encjson.AppendArrayStart(encjson.AppendKey(buf, a.key))
	for _, val := range a.vals {
		buf = encjson.AppendString(buf, val)
	}
	return encjson.AppendArrayEnd(buf)
}

func (a *Strings) String() string {
	return fmt.Sprintf("Strings{%q: %s}", a.key, a.ValueString())
}

func (a *Strings) Len() int { return len(a.vals) }

// Error

type Error struct {
	key string
	val error
}

func NewError(key string, val error) *Error {
	a := errorPool.GetOrNew()
	a.key = key
	a.val = val
	return a
}

func (a *Error) Clone() Attrib {
	return NewError(a.key, a.val)
}

func (a *Error) Free() {
	errorPool.ZeroAndPutBack(a)
}

func (a *Error) Key() string { return a.key }
func (a *Error) Value() any  { return a.val }
func (a *Error) ValueString() string {
	if a.val == nil {
		return "<nil>"
	}
	return a.val.Error()
}

func (a *Error) Log(m *Message) {
	m.Error(a.key, a.val)
}

func (a *Error) AppendJSON(buf []byte) []byte {
	buf = encjson.AppendKey(buf, a.key)
	if a.val == nil {
		return encjson.AppendNull(buf)
	}
	return encjson.AppendString(buf, a.val.Error())
}

func (a *Error) String() string {
	return fmt.Sprintf("Error{%q: %q}", a.key, a.ValueString())
}

type Errors struct {
	key  string
	vals []error
}

func NewErrors(key string, vals []error) *Errors {
	a := errorsPool.GetOrNew()
	a.key = key
	a.vals = vals
	return a
}

func NewErrorsCopy(key string, vals []error) *Errors {
	return NewErrors(key, slices.Clone(vals))
}

func (a *Errors) Clone() Attrib {
	return NewErrors(a.key, a.vals)
}

func (a *Errors) Free() {
	errorsPool.ZeroAndPutBack(a)
}

func (a *Errors) Key() string { return a.key }
func (a *Errors) Value() any  { return a.vals }

func (a *Errors) ValueString() string {
	// errors.Join returns nil for no or only nil errors
	if err := errors.Join(a.vals...); err != nil {
		return err.Error()
	}
	return "<nil>"
}

func (a *Errors) Log(m *Message) {
	m.Errors(a.key, a.vals)
}

func (a *Errors) AppendJSON(buf []byte) []byte {
	buf = encjson.AppendArrayStart(encjson.AppendKey(buf, a.key))
	for _, val := range a.vals {
		if val == nil {
			buf = encjson.AppendNull(buf)
		} else {
			buf = encjson.AppendString(buf, val.Error())
		}
	}
	return encjson.AppendArrayEnd(buf)
}

func (a *Errors) String() string {
	return fmt.Sprintf("Errors{%q: %q}", a.key, a.ValueString())
}

func (a *Errors) Len() int { return len(a.vals) }

// Time

type Time struct {
	key string
	val time.Time
}

func NewTime(key string, val time.Time) *Time {
	a := timePool.GetOrNew()
	a.key = key
	a.val = val
	return a
}

func (a *Time) Clone() Attrib {
	return NewTime(a.key, a.val)
}

func (a *Time) Free() {
	timePool.ZeroAndPutBack(a)
}

func (a *Time) Key() string         { return a.key }
func (a *Time) Value() any          { return a.val }
func (a *Time) ValueString() string { return a.val.Format(DefaultTimeFormat) }

func (a *Time) Log(m *Message) {
	m.Time(a.key, a.val)
}

func (a *Time) AppendJSON(buf []byte) []byte {
	return encjson.AppendTime(encjson.AppendKey(buf, a.key), a.val, DefaultTimeFormat)
}

func (a *Time) String() string {
	return fmt.Sprintf("Time{%q: %s}", a.key, a.val.Format(DefaultTimeFormat))
}

type Times struct {
	key  string
	vals []time.Time
}

func NewTimes(key string, vals []time.Time) *Times {
	a := timesPool.GetOrNew()
	a.key = key
	a.vals = vals
	return a
}

func NewTimesCopy(key string, vals []time.Time) *Times {
	if vals == nil {
		return NewTimes(key, nil)
	}
	times := make([]time.Time, len(vals))
	copy(times, vals)
	return NewTimes(key, times)
}

func (a *Times) Clone() Attrib {
	return NewTimes(a.key, a.vals)
}

func (a *Times) Free() {
	timesPool.ZeroAndPutBack(a)
}

func (a *Times) Key() string         { return a.key }
func (a *Times) Value() any          { return a.vals }
func (a *Times) ValueString() string { return fmt.Sprintf("%#v", a.vals) }

func (a *Times) Log(m *Message) {
	m.Times(a.key, a.vals)
}

func (a *Times) AppendJSON(buf []byte) []byte {
	buf = encjson.AppendArrayStart(encjson.AppendKey(buf, a.key))
	for _, val := range a.vals {
		buf = encjson.AppendTime(buf, val, DefaultTimeFormat)
	}
	return encjson.AppendArrayEnd(buf)
}

func (a *Times) String() string {
	return fmt.Sprintf("Times{%q: %s}", a.key, a.ValueString())
}

func (a *Times) Len() int { return len(a.vals) }

// UUID

type UUID struct {
	key string
	val [16]byte
}

// NewUUID creates a new UUID attribute with the passed key and value.
//
// See NewUUIDv4 and UUIDv4 for creating a new random UUID value.
func NewUUID(key string, val [16]byte) *UUID {
	a := uuidPool.GetOrNew()
	a.key = key
	a.val = val
	return a
}

// NewUUIDv4 creates a new UUID attribute with the passed key and a random version 4 UUID value.
func NewUUIDv4(key string) *UUID {
	return NewUUID(key, UUIDv4())
}

func (a *UUID) Clone() Attrib {
	return NewUUID(a.key, a.val)
}

func (a *UUID) Free() {
	uuidPool.ZeroAndPutBack(a)
}

func (a *UUID) Key() string         { return a.key }
func (a *UUID) Value() any          { return a.val }
func (a *UUID) ValueString() string { return FormatUUID(a.val) }
func (a *UUID) ValueUUID() [16]byte { return a.val }

func (a *UUID) Log(m *Message) {
	m.UUID(a.key, a.val)
}

func (a *UUID) AppendJSON(buf []byte) []byte {
	buf = encjson.AppendKey(buf, a.key)
	if IsNilUUID(a.val) {
		// Like Message.UUID
		return encjson.AppendNull(buf)
	}
	return encjson.AppendUUID(buf, a.val)
}

func (a *UUID) String() string {
	return fmt.Sprintf("UUID{%q: %s}", a.key, a.ValueString())
}

type UUIDs struct {
	key  string
	vals [][16]byte
}

func NewUUIDs(key string, vals [][16]byte) *UUIDs {
	a := uuidsPool.GetOrNew()
	a.key = key
	a.vals = vals
	return a
}

func NewUUIDsCopy[T ~[16]byte](key string, vals []T) *UUIDs {
	if vals == nil {
		return NewUUIDs(key, nil)
	}
	uuids := make([][16]byte, len(vals))
	for i, val := range vals {
		uuids[i] = val
	}
	return NewUUIDs(key, uuids)
}

func (a *UUIDs) Clone() Attrib {
	return NewUUIDs(a.key, a.vals)
}

func (a *UUIDs) Free() {
	uuidsPool.ZeroAndPutBack(a)
}

func (a *UUIDs) Key() string { return a.key }
func (a *UUIDs) Value() any  { return a.vals }
func (a *UUIDs) ValueString() string {
	var b strings.Builder
	b.WriteByte('[')
	for i := range a.vals {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(FormatUUID(a.vals[i]))
	}
	b.WriteByte(']')
	return b.String()
}
func (a *UUIDs) ValueUUIDs() [][16]byte { return a.vals }

func (a *UUIDs) Log(m *Message) {
	m.UUIDs(a.key, a.vals)
}

func (a *UUIDs) AppendJSON(buf []byte) []byte {
	buf = encjson.AppendArrayStart(encjson.AppendKey(buf, a.key))
	for _, val := range a.vals {
		if IsNilUUID(val) {
			// Like Message.UUIDs
			buf = encjson.AppendNull(buf)
		} else {
			buf = encjson.AppendUUID(buf, val)
		}
	}
	return encjson.AppendArrayEnd(buf)
}

func (a *UUIDs) String() string {
	return fmt.Sprintf("UUIDs{%q: %s}", a.key, a.ValueString())
}

func (a *UUIDs) Len() int { return len(a.vals) }

// JSON

type JSON struct {
	key string
	val json.RawMessage
}

func NewJSON(key string, val json.RawMessage) *JSON {
	a := jsonPool.GetOrNew()
	a.key = key
	a.val = val
	return a
}

func (a *JSON) Clone() Attrib {
	return NewJSON(a.key, a.val)
}

func (a *JSON) Free() {
	jsonPool.ZeroAndPutBack(a)
}

func (a *JSON) Key() string                { return a.key }
func (a *JSON) Value() any                 { return a.val }
func (a *JSON) ValueString() string        { return string(a.val) }
func (a *JSON) ValueJSON() json.RawMessage { return a.val }

func (a *JSON) Log(m *Message) {
	m.JSON(a.key, a.val)
}

func (a *JSON) AppendJSON(buf []byte) []byte {
	buf = encjson.AppendKey(buf, a.key)
	if len(a.val) == 0 {
		// An empty value is no valid JSON, log it as null like Message.JSON
		return encjson.AppendNull(buf)
	}
	return append(buf, a.val...)
}

func (a *JSON) String() string {
	return fmt.Sprintf("JSON{%q: %s}", a.key, a.ValueString())
}
