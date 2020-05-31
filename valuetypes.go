package golog

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

// Any

type AnyValue struct {
	Key string
	Val interface{}
}

func NewAnyValue(key string, val interface{}) *AnyValue {
	return &AnyValue{Key: key, Val: val}
}

func (v *AnyValue) Name() string { return v.Key }

func (v *AnyValue) Log(m *Message) {
	m.Any(v.Key, v.Val)
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
