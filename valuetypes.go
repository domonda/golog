package golog

import "fmt"

// Nil

type NilValue struct {
	Key string
}

func NewNilValue(key string) *NilValue {
	return &NilValue{Key: key}
}

func (v *NilValue) Name() string { return v.Key }

func (v *NilValue) Log(m *Message) {
	m.Nil(v.Key)
}

func (v *NilValue) String() string {
	if v == nil {
		return "<nil>"
	}
	return fmt.Sprintf("NilValue{%s}", v.Key)
}

// Any

type AnyValue struct {
	Key string
	Val any
}

func NewAnyValue(key string, val any) *AnyValue {
	return &AnyValue{Key: key, Val: val}
}

func (v *AnyValue) Name() string { return v.Key }

func (v *AnyValue) Log(m *Message) {
	m.Any(v.Key, v.Val)
}

func (v *AnyValue) String() string {
	if v == nil {
		return "<nil>"
	}
	return fmt.Sprintf("AnyValue{%s:%v}", v.Key, v.Val)
}

// Bool

type BoolValue struct {
	Key string
	Val bool
}

func NewBoolValue(key string, val bool) *BoolValue {
	return &BoolValue{Key: key, Val: val}
}

func (v *BoolValue) Name() string { return v.Key }

func (v *BoolValue) Log(m *Message) {
	m.Bool(v.Key, v.Val)
}

func (v *BoolValue) String() string {
	if v == nil {
		return "<nil>"
	}
	return fmt.Sprintf("BoolValue{%s:%v}", v.Key, v.Val)
}

type BoolsValue struct {
	Key  string
	Vals []bool
}

func NewBoolsValue(key string, vals ...bool) *BoolsValue {
	return &BoolsValue{Key: key, Vals: vals}
}

func (v *BoolsValue) Name() string { return v.Key }

func (v *BoolsValue) Log(m *Message) {
	m.Bools(v.Key, v.Vals)
}

func (v *BoolsValue) String() string {
	if v == nil {
		return "<nil>"
	}
	return fmt.Sprintf("BoolsValue{%s:%v}", v.Key, v.Vals)
}

// Int

type IntValue struct {
	Key string
	Val int64
}

func NewIntValue(key string, val int64) *IntValue {
	return &IntValue{Key: key, Val: val}
}

func (v *IntValue) Name() string { return v.Key }

func (v *IntValue) Log(m *Message) {
	m.Int64(v.Key, v.Val)
}

func (v *IntValue) String() string {
	if v == nil {
		return "<nil>"
	}
	return fmt.Sprintf("IntValue{%s:%v}", v.Key, v.Val)
}

type IntsValue struct {
	Key  string
	Vals []int64
}

func NewIntsValue(key string, vals ...int64) *IntsValue {
	return &IntsValue{Key: key, Vals: vals}
}

func (v *IntsValue) Name() string { return v.Key }

func (v *IntsValue) Log(m *Message) {
	m.Int64s(v.Key, v.Vals)
}

func (v *IntsValue) String() string {
	if v == nil {
		return "<nil>"
	}
	return fmt.Sprintf("IntsValue{%s:%v}", v.Key, v.Vals)
}

// Uint

type UintValue struct {
	Key string
	Val uint64
}

func NewUintValue(key string, val uint64) *UintValue {
	return &UintValue{Key: key, Val: val}
}

func (v *UintValue) Name() string { return v.Key }

func (v *UintValue) Log(m *Message) {
	m.Uint64(v.Key, v.Val)
}

func (v *UintValue) String() string {
	if v == nil {
		return "<nil>"
	}
	return fmt.Sprintf("UintValue{%s:%v}", v.Key, v.Val)
}

type UintsValue struct {
	Key  string
	Vals []uint64
}

func NewUintsValue(key string, vals ...uint64) *UintsValue {
	return &UintsValue{Key: key, Vals: vals}
}

func (v *UintsValue) Name() string { return v.Key }

func (v *UintsValue) Log(m *Message) {
	m.Uint64s(v.Key, v.Vals)
}

func (v *UintsValue) String() string {
	if v == nil {
		return "<nil>"
	}
	return fmt.Sprintf("UintsValue{%s:%v}", v.Key, v.Vals)
}

// Float

type FloatValue struct {
	Key string
	Val float64
}

func NewFloatValue(key string, val float64) *FloatValue {
	return &FloatValue{Key: key, Val: val}
}

func (v *FloatValue) Name() string { return v.Key }

func (v *FloatValue) Log(m *Message) {
	m.Float(v.Key, v.Val)
}

func (v *FloatValue) String() string {
	if v == nil {
		return "<nil>"
	}
	return fmt.Sprintf("FloatValue{%s:%f}", v.Key, v.Val)
}

type FloatsValue struct {
	Key  string
	Vals []float64
}

func NewFloatsValue(key string, vals ...float64) *FloatsValue {
	return &FloatsValue{Key: key, Vals: vals}
}

func (v *FloatsValue) Name() string { return v.Key }

func (v *FloatsValue) Log(m *Message) {
	m.Floats(v.Key, v.Vals)
}

func (v *FloatsValue) String() string {
	if v == nil {
		return "<nil>"
	}
	return fmt.Sprintf("FloatsValue{%s:%v}", v.Key, v.Vals)
}

// String

type StringValue struct {
	Key string
	Val string
}

func NewStringValue(key string, val string) *StringValue {
	return &StringValue{Key: key, Val: val}
}

func (v *StringValue) Name() string { return v.Key }

func (v *StringValue) Log(m *Message) {
	m.Str(v.Key, v.Val)
}

func (v *StringValue) String() string {
	if v == nil {
		return "<nil>"
	}
	return fmt.Sprintf("StringValue{%s:%s}", v.Key, v.Val)
}

type StringsValue struct {
	Key  string
	Vals []string
}

func NewStringsValue(key string, vals ...string) *StringsValue {
	return &StringsValue{Key: key, Vals: vals}
}

func (v *StringsValue) Name() string { return v.Key }

func (v *StringsValue) Log(m *Message) {
	m.Strs(v.Key, v.Vals)
}

func (v *StringsValue) String() string {
	if v == nil {
		return "<nil>"
	}
	return fmt.Sprintf("StringsValue{%s:%v}", v.Key, v.Vals)
}

// Error

type ErrorValue struct {
	Key string
	Val error
}

func NewErrorValue(key string, val error) *ErrorValue {
	return &ErrorValue{Key: key, Val: val}
}

func (v *ErrorValue) Name() string { return v.Key }

func (v *ErrorValue) Log(m *Message) {
	m.Error(v.Key, v.Val)
}

func (v *ErrorValue) String() string {
	if v == nil {
		return "<nil>"
	}
	return fmt.Sprintf("ErrorValue{%s:%s}", v.Key, v.Val)
}

type ErrorsValue struct {
	Key  string
	Vals []error
}

func NewErrorsValue(key string, vals ...error) *ErrorsValue {
	return &ErrorsValue{Key: key, Vals: vals}
}

func (v *ErrorsValue) Name() string { return v.Key }

func (v *ErrorsValue) Log(m *Message) {
	m.Errors(v.Key, v.Vals)
}

func (v *ErrorsValue) String() string {
	if v == nil {
		return "<nil>"
	}
	return fmt.Sprintf("ErrorsValue{%s:%v}", v.Key, v.Vals)
}

// UUID

type UUIDValue struct {
	Key string
	Val [16]byte
}

func NewUUIDValue(key string, val [16]byte) *UUIDValue {
	return &UUIDValue{Key: key, Val: val}
}

func (v *UUIDValue) Name() string { return v.Key }

func (v *UUIDValue) Log(m *Message) {
	m.UUID(v.Key, v.Val)
}

func (v *UUIDValue) String() string {
	if v == nil {
		return "<nil>"
	}
	return fmt.Sprintf("UUIDValue{%s:%s}", v.Key, v.Val)
}

type UUIDsValue struct {
	Key  string
	Vals [][16]byte
}

func NewUUIDsValue(key string, vals ...[16]byte) *UUIDsValue {
	return &UUIDsValue{Key: key, Vals: vals}
}

func (v *UUIDsValue) Name() string { return v.Key }

func (v *UUIDsValue) Log(m *Message) {
	m.UUIDs(v.Key, v.Vals)
}

func (v *UUIDsValue) String() string {
	if v == nil {
		return "<nil>"
	}
	return fmt.Sprintf("UUIDsValue{%s:%v}", v.Key, v.Vals)
}

// JSON

type JSONValue struct {
	Key string
	Val []byte
}

func NewJSONValue(key string, val []byte) *JSONValue {
	return &JSONValue{Key: key, Val: val}
}

func (v *JSONValue) Name() string { return v.Key }

func (v *JSONValue) Log(m *Message) {
	m.JSON(v.Key, v.Val)
}

func (v *JSONValue) String() string {
	if v == nil {
		return "<nil>"
	}
	return fmt.Sprintf("JSONValue{%s:%s}", v.Key, v.Val)
}
