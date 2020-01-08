package golog

// NamedValue is an interface that allows types
// to log themselves.
type NamedValue interface {
	// Log logs the object to a message.
	Log(message *Message)
}

// NamedValueFunc implements NamedValue with a function
type NamedValueFunc func(*Message)

func (f NamedValueFunc) Log(message *Message) { f(message) }

// Nil

type NilNamedValue struct {
	Key string
}

func (v *NilNamedValue) Log(message *Message) {
	message.Nil(v.Key)
}

// Bool

type BoolNamedValue struct {
	Key string
	Val bool
}

func (v *BoolNamedValue) Log(message *Message) {
	message.Bool(v.Key, v.Val)
}

type BoolsNamedValue struct {
	Key  string
	Vals []bool
}

func (v *BoolsNamedValue) Log(message *Message) {
	message.Bools(v.Key, v.Vals)
}

// Int

type IntNamedValue struct {
	Key string
	Val int64
}

func (v *IntNamedValue) Log(message *Message) {
	message.Int64(v.Key, v.Val)
}

type IntsNamedValue struct {
	Key  string
	Vals []int64
}

func (v *IntsNamedValue) Log(message *Message) {
	message.Int64s(v.Key, v.Vals)
}

// Uint

type UintNamedValue struct {
	Key string
	Val uint64
}

func (v *UintNamedValue) Log(message *Message) {
	message.Uint64(v.Key, v.Val)
}

type UintsNamedValue struct {
	Key  string
	Vals []uint64
}

func (v *UintsNamedValue) Log(message *Message) {
	message.Uint64s(v.Key, v.Vals)
}

// Float

type FloatNamedValue struct {
	Key string
	Val float64
}

func (v *FloatNamedValue) Log(message *Message) {
	message.Float(v.Key, v.Val)
}

type FloatsNamedValue struct {
	Key  string
	Vals []float64
}

func (v *FloatsNamedValue) Log(message *Message) {
	message.Floats(v.Key, v.Vals)
}

// String

type StringNamedValue struct {
	Key string
	Val string
}

func (v *StringNamedValue) Log(message *Message) {
	message.Str(v.Key, v.Val)
}

type StringsNamedValue struct {
	Key  string
	Vals []string
}

func (v *StringsNamedValue) Log(message *Message) {
	message.Strs(v.Key, v.Vals)
}

// Error

type ErrorNamedValue struct {
	Key string
	Val error
}

func (v *ErrorNamedValue) Log(message *Message) {
	message.Error(v.Key, v.Val)
}

type ErrorsNamedValue struct {
	Key  string
	Vals []error
}

func (v *ErrorsNamedValue) Log(message *Message) {
	message.Errors(v.Key, v.Vals)
}

// UUID

type UUIDNamedValue struct {
	Key string
	Val [16]byte
}

func (v *UUIDNamedValue) Log(message *Message) {
	message.UUID(v.Key, v.Val)
}

type UUIDsNamedValue struct {
	Key  string
	Vals [][16]byte
}

func (v *UUIDsNamedValue) Log(message *Message) {
	message.UUIDs(v.Key, v.Vals)
}

// JSON

type JSONNamedValue struct {
	Key string
	Val []byte
}

func (v *JSONNamedValue) Log(message *Message) {
	message.JSON(v.Key, v.Val)
}
