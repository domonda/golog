package golog

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// Attrib extends the Loggable interface and allows
// attributes to log themselves and be referenced by a key.
type Attrib interface {
	Loggable
	fmt.Stringer

	// GetKey returns the attribute key
	GetKey() string

	// GetVal returns the attribute value
	GetVal() any

	// GetValString returns the attribute value
	// formatted as string
	GetValString() string
}

type SliceAttrib interface {
	Attrib

	Len() int
}

// Attrib implementations
var (
	_ Attrib      = Nil{}
	_ Attrib      = Any{}
	_ Attrib      = Bool{}
	_ Attrib      = Bools{}
	_ SliceAttrib = Bools{}
	_ Attrib      = Int{}
	_ Attrib      = Ints{}
	_ SliceAttrib = Ints{}
	_ Attrib      = Uint{}
	_ Attrib      = Uints{}
	_ SliceAttrib = Uints{}
	_ Attrib      = Float{}
	_ Attrib      = Floats{}
	_ SliceAttrib = Floats{}
	_ Attrib      = String{}
	_ Attrib      = Strings{}
	_ SliceAttrib = Strings{}
	_ Attrib      = Error{}
	_ Attrib      = Errors{}
	_ SliceAttrib = Errors{}
	_ Attrib      = UUID{}
	_ Attrib      = UUIDs{}
	_ SliceAttrib = UUIDs{}
	_ Attrib      = JSON{}
)

// Nil

type Nil struct {
	Key string
}

func (a Nil) GetKey() string       { return a.Key }
func (a Nil) GetVal() any          { return nil }
func (a Nil) GetValString() string { return "<nil>" }

func (a Nil) Log(m *Message) {
	m.Nil(a.Key)
}

func (a Nil) String() string {
	return fmt.Sprintf("Nil{%s}", a.Key)
}

// Any

type Any struct {
	Key string
	Val any
}

func (a Any) GetKey() string       { return a.Key }
func (a Any) GetVal() any          { return a.Val }
func (a Any) GetValString() string { return fmt.Sprintf("%#v", a.Val) }

func (a Any) Log(m *Message) {
	m.Any(a.Key, a.Val)
}

func (a Any) String() string {
	return fmt.Sprintf("Any{%q: %s}", a.Key, a.GetValString())
}

// Bool

type Bool struct {
	Key string
	Val bool
}

func (a Bool) GetKey() string       { return a.Key }
func (a Bool) GetVal() any          { return a.Val }
func (a Bool) GetValString() string { return fmt.Sprintf("%#v", a.Val) }

func (a Bool) Log(m *Message) {
	m.Bool(a.Key, a.Val)
}

func (a Bool) String() string {
	return fmt.Sprintf("Bool{%q: %s}", a.Key, a.GetValString())
}

type Bools struct {
	Key  string
	Vals []bool
}

func (a Bools) GetKey() string       { return a.Key }
func (a Bools) GetVal() any          { return a.Vals }
func (a Bools) GetValString() string { return fmt.Sprintf("%#v", a.Vals) }

func (a Bools) Log(m *Message) {
	m.Bools(a.Key, a.Vals)
}

func (a Bools) String() string {
	return fmt.Sprintf("Bools{%q: %s}", a.Key, a.GetValString())
}

func (a Bools) Len() int { return len(a.Vals) }

// Int

type Int struct {
	Key string
	Val int64
}

func (a Int) GetKey() string       { return a.Key }
func (a Int) GetVal() any          { return a.Val }
func (a Int) GetValString() string { return fmt.Sprintf("%#v", a.Val) }

func (a Int) Log(m *Message) {
	m.Int64(a.Key, a.Val)
}

func (a Int) String() string {
	return fmt.Sprintf("Int{%q: %s}", a.Key, a.GetValString())
}

type Ints struct {
	Key  string
	Vals []int64
}

func (a Ints) GetKey() string       { return a.Key }
func (a Ints) GetVal() any          { return a.Vals }
func (a Ints) GetValString() string { return fmt.Sprintf("%#v", a.Vals) }

func (a Ints) Log(m *Message) {
	m.Int64s(a.Key, a.Vals)
}

func (a Ints) String() string {
	return fmt.Sprintf("Ints{%q: %s}", a.Key, a.GetValString())
}

func (a Ints) Len() int { return len(a.Vals) }

// Uint

type Uint struct {
	Key string
	Val uint64
}

func (a Uint) GetKey() string       { return a.Key }
func (a Uint) GetVal() any          { return a.Val }
func (a Uint) GetValString() string { return fmt.Sprintf("%#v", a.Val) }

func (a Uint) Log(m *Message) {
	m.Uint64(a.Key, a.Val)
}

func (a Uint) String() string {
	return fmt.Sprintf("Uint{%q: %s}", a.Key, a.GetValString())
}

type Uints struct {
	Key  string
	Vals []uint64
}

func (a Uints) GetKey() string       { return a.Key }
func (a Uints) GetVal() any          { return a.Vals }
func (a Uints) GetValString() string { return fmt.Sprintf("%#v", a.Vals) }

func (a Uints) Log(m *Message) {
	m.Uint64s(a.Key, a.Vals)
}

func (a Uints) String() string {
	return fmt.Sprintf("Uints{%q: %s}", a.Key, a.GetValString())
}

func (a Uints) Len() int { return len(a.Vals) }

// Float

type Float struct {
	Key string
	Val float64
}

func (a Float) GetKey() string       { return a.Key }
func (a Float) GetVal() any          { return a.Val }
func (a Float) GetValString() string { return fmt.Sprintf("%#v", a.Val) }

func (a Float) Log(m *Message) {
	m.Float(a.Key, a.Val)
}

func (a Float) String() string {
	return fmt.Sprintf("Float{%q: %s}", a.Key, a.GetValString())
}

type Floats struct {
	Key  string
	Vals []float64
}

func (a Floats) GetKey() string       { return a.Key }
func (a Floats) GetVal() any          { return a.Vals }
func (a Floats) GetValString() string { return fmt.Sprintf("%#v", a.Vals) }

func (a Floats) Log(m *Message) {
	m.Floats(a.Key, a.Vals)
}

func (a Floats) String() string {
	return fmt.Sprintf("Floats{%q: %s}", a.Key, a.GetValString())
}

func (a Floats) Len() int { return len(a.Vals) }

// String

type String struct {
	Key string
	Val string
}

func (a String) GetKey() string       { return a.Key }
func (a String) GetVal() any          { return a.Val }
func (a String) GetValString() string { return a.Val }

func (a String) Log(m *Message) {
	m.Str(a.Key, a.Val)
}

func (a String) String() string {
	return fmt.Sprintf("String{%q: %q}", a.Key, a.Val)
}

type Strings struct {
	Key  string
	Vals []string
}

func (a Strings) GetKey() string       { return a.Key }
func (a Strings) GetVal() any          { return a.Vals }
func (a Strings) GetValString() string { return fmt.Sprintf("%#v", a.Vals) }

func (a Strings) Log(m *Message) {
	m.Strs(a.Key, a.Vals)
}

func (a Strings) String() string {
	return fmt.Sprintf("Strings{%q: %s}", a.Key, a.GetValString())
}

func (a Strings) Len() int { return len(a.Vals) }

// Error

type Error struct {
	Key string
	Val error
}

func (a Error) GetKey() string { return a.Key }
func (a Error) GetVal() any    { return a.Val }
func (a Error) GetValString() string {
	if a.Val == nil {
		return "<nil>"
	}
	return a.Val.Error()
}

func (a Error) Log(m *Message) {
	m.Error(a.Key, a.Val)
}

func (a Error) String() string {
	return fmt.Sprintf("Error{%q: %q}", a.Key, a.GetValString())
}

type Errors struct {
	Key  string
	Vals []error
}

func (a Errors) GetKey() string { return a.Key }
func (a Errors) GetVal() any    { return a.Vals }

func (a Errors) GetValString() string {
	if len(a.Vals) == 0 {
		return "<nil>"
	}
	return errors.Join(a.Vals...).Error()
}

func (a Errors) Log(m *Message) {
	m.Errors(a.Key, a.Vals)
}

func (a Errors) String() string {
	return fmt.Sprintf("Errors{%q: %q}", a.Key, a.GetValString())
}

func (a Errors) Len() int { return len(a.Vals) }

// UUID

type UUID struct {
	Key string
	Val [16]byte
}

func (a UUID) GetKey() string       { return a.Key }
func (a UUID) GetVal() any          { return a.Val }
func (a UUID) GetValString() string { return FormatUUID(a.Val) }

func (a UUID) Log(m *Message) {
	m.UUID(a.Key, a.Val)
}

func (a UUID) String() string {
	return fmt.Sprintf("UUID{%q: %s}", a.Key, a.GetValString())
}

type UUIDs struct {
	Key  string
	Vals [][16]byte
}

func (a UUIDs) GetKey() string { return a.Key }
func (a UUIDs) GetVal() any    { return a.Vals }

func (a UUIDs) GetValString() string {
	var b strings.Builder
	b.WriteByte('[')
	for i := range a.Vals {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(FormatUUID(a.Vals[i]))
	}
	b.WriteByte(']')
	return b.String()
}

func (a UUIDs) Log(m *Message) {
	m.UUIDs(a.Key, a.Vals)
}

func (a UUIDs) String() string {
	return fmt.Sprintf("UUIDs{%q: %s}", a.Key, a.GetValString())
}

func (a UUIDs) Len() int { return len(a.Vals) }

// JSON

type JSON struct {
	Key string
	Val json.RawMessage
}

func (a JSON) GetKey() string       { return a.Key }
func (a JSON) GetVal() any          { return a.Val }
func (a JSON) GetValString() string { return string(a.Val) }

func (a JSON) Log(m *Message) {
	m.JSON(a.Key, a.Val)
}

func (a JSON) String() string {
	return fmt.Sprintf("JSON{%q: %s}", a.Key, a.GetValString())
}

// // Bytes

// type Bytes struct {
// 	Key string
// 	Val []byte
// }

// func (a Bytes) GetKey() string { return a.Key }

// func (a Bytes) Log(m *Message) {
// 	m.Bytes(a.Key, a.Val)
// }

// func (a Bytes) String() string {
// 	return fmt.Sprintf("Bytes{%q: %x}", a.Key, a.Val)
// }
