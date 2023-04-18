package golog

import (
	"encoding/json"
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
}

// Attrib implementations
var (
	_ Attrib = Nil{}
	_ Attrib = Any{}
	_ Attrib = Bool{}
	_ Attrib = Bools{}
	_ Attrib = Int{}
	_ Attrib = Ints{}
	_ Attrib = Uint{}
	_ Attrib = Uints{}
	_ Attrib = Float{}
	_ Attrib = Floats{}
	_ Attrib = String{}
	_ Attrib = Strings{}
	_ Attrib = Error{}
	_ Attrib = Errors{}
	_ Attrib = UUID{}
	_ Attrib = UUIDs{}
	_ Attrib = JSON{}
)

// Nil

type Nil struct {
	Key string
}

func (a Nil) GetKey() string { return a.Key }

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

func (a Any) GetKey() string { return a.Key }

func (a Any) Log(m *Message) {
	m.Any(a.Key, a.Val)
}

func (a Any) String() string {
	return fmt.Sprintf("Any{%q: %#v}", a.Key, a.Val)
}

// Bool

type Bool struct {
	Key string
	Val bool
}

func (a Bool) GetKey() string { return a.Key }

func (a Bool) Log(m *Message) {
	m.Bool(a.Key, a.Val)
}

func (a Bool) String() string {
	return fmt.Sprintf("Bool{%q: %#v}", a.Key, a.Val)
}

type Bools struct {
	Key  string
	Vals []bool
}

func (a Bools) GetKey() string { return a.Key }

func (a Bools) Log(m *Message) {
	m.Bools(a.Key, a.Vals)
}

func (a Bools) String() string {
	return fmt.Sprintf("Bools{%q: %#v}", a.Key, a.Vals)
}

// Int

type Int struct {
	Key string
	Val int64
}

func (a Int) GetKey() string { return a.Key }

func (a Int) Log(m *Message) {
	m.Int64(a.Key, a.Val)
}

func (a Int) String() string {
	return fmt.Sprintf("Int{%q: %#v}", a.Key, a.Val)
}

type Ints struct {
	Key  string
	Vals []int64
}

func (a Ints) GetKey() string { return a.Key }

func (a Ints) Log(m *Message) {
	m.Int64s(a.Key, a.Vals)
}

func (a Ints) String() string {
	return fmt.Sprintf("Ints{%q: %#v}", a.Key, a.Vals)
}

// Uint

type Uint struct {
	Key string
	Val uint64
}

func (a Uint) GetKey() string { return a.Key }

func (a Uint) Log(m *Message) {
	m.Uint64(a.Key, a.Val)
}

func (a Uint) String() string {
	return fmt.Sprintf("Uint{%q: %#v}", a.Key, a.Val)
}

type Uints struct {
	Key  string
	Vals []uint64
}

func (a Uints) GetKey() string { return a.Key }

func (a Uints) Log(m *Message) {
	m.Uint64s(a.Key, a.Vals)
}

func (a Uints) String() string {
	return fmt.Sprintf("Uints{%q: %#v}", a.Key, a.Vals)
}

// Float

type Float struct {
	Key string
	Val float64
}

func (a Float) GetKey() string { return a.Key }

func (a Float) Log(m *Message) {
	m.Float(a.Key, a.Val)
}

func (a Float) String() string {
	return fmt.Sprintf("Float{%q: %f}", a.Key, a.Val)
}

type Floats struct {
	Key  string
	Vals []float64
}

func (a Floats) GetKey() string { return a.Key }

func (a Floats) Log(m *Message) {
	m.Floats(a.Key, a.Vals)
}

func (a Floats) String() string {
	return fmt.Sprintf("Floats{%q: %#v}", a.Key, a.Vals)
}

// String

type String struct {
	Key string
	Val string
}

func (a String) GetKey() string { return a.Key }

func (a String) Log(m *Message) {
	m.Str(a.Key, a.Val)
}

func (a String) String() string {
	return fmt.Sprintf("String{%q: %#v}", a.Key, a.Val)
}

type Strings struct {
	Key  string
	Vals []string
}

func (a Strings) GetKey() string { return a.Key }

func (a Strings) Log(m *Message) {
	m.Strs(a.Key, a.Vals)
}

func (a Strings) String() string {
	return fmt.Sprintf("Strings{%q: %#v}", a.Key, a.Vals)
}

// Error

type Error struct {
	Key string
	Val error
}

func (a Error) GetKey() string { return a.Key }

func (a Error) Log(m *Message) {
	m.Error(a.Key, a.Val)
}

func (a Error) String() string {
	return fmt.Sprintf("Error{%q: %#v}", a.Key, a.Val)
}

type Errors struct {
	Key  string
	Vals []error
}

func (a Errors) GetKey() string { return a.Key }

func (a Errors) Log(m *Message) {
	m.Errors(a.Key, a.Vals)
}

func (a Errors) String() string {
	return fmt.Sprintf("Errors{%q: %#v}", a.Key, a.Vals)
}

// UUID

type UUID struct {
	Key string
	Val [16]byte
}

func (a UUID) GetKey() string { return a.Key }

func (a UUID) Log(m *Message) {
	m.UUID(a.Key, a.Val)
}

func (a UUID) String() string {
	return fmt.Sprintf("UUID{%q: %s}", a.Key, FormatUUID(a.Val))
}

type UUIDs struct {
	Key  string
	Vals [][16]byte
}

func (a UUIDs) GetKey() string { return a.Key }

func (a UUIDs) Log(m *Message) {
	m.UUIDs(a.Key, a.Vals)
}

func (a UUIDs) String() string {
	var b strings.Builder
	b.WriteByte('[')
	for i := range a.Vals {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(FormatUUID(a.Vals[i]))
	}
	b.WriteByte(']')
	return fmt.Sprintf("UUIDs{%q: %s}", a.Key, b.String())
}

// JSON

type JSON struct {
	Key string
	Val json.RawMessage
}

func (a JSON) GetKey() string { return a.Key }

func (a JSON) Log(m *Message) {
	m.JSON(a.Key, a.Val)
}

func (a JSON) String() string {
	return fmt.Sprintf("JSON{%q: %s}", a.Key, a.Val)
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
