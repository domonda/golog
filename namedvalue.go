package golog

// NamedValue is an interface that allows types
// to log themselves.
type NamedValue interface {
	// Name returns the name of the value
	Name() string

	// Log logs the object to a message.
	Log(message *Message)
}

// MergeNamedValues merges a and b so that only one
// value with a given name is in the result.
// Order is preserved, values from a are removed if
// they are also present with the same name in b.
// Wihtout name collisions, the result is identical to append(a, b).
func MergeNamedValues(a, b []NamedValue) []NamedValue {
	if len(a) == 0 && len(b) == 0 {
		return nil
	}

	c := make([]NamedValue, len(a), len(a)+len(b))
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

// Nil

type NilNamedValue struct {
	Key string
}

func (v *NilNamedValue) Name() string { return v.Key }

func (v *NilNamedValue) Log(message *Message) {
	message.Nil(v.Key)
}

// Bool

type BoolNamedValue struct {
	Key string
	Val bool
}

func (v *BoolNamedValue) Name() string { return v.Key }

func (v *BoolNamedValue) Log(message *Message) {
	message.Bool(v.Key, v.Val)
}

type BoolsNamedValue struct {
	Key  string
	Vals []bool
}

func (v *BoolsNamedValue) Name() string { return v.Key }

func (v *BoolsNamedValue) Log(message *Message) {
	message.Bools(v.Key, v.Vals)
}

// Int

type IntNamedValue struct {
	Key string
	Val int64
}

func (v *IntNamedValue) Name() string { return v.Key }

func (v *IntNamedValue) Log(message *Message) {
	message.Int64(v.Key, v.Val)
}

type IntsNamedValue struct {
	Key  string
	Vals []int64
}

func (v *IntsNamedValue) Name() string { return v.Key }

func (v *IntsNamedValue) Log(message *Message) {
	message.Int64s(v.Key, v.Vals)
}

// Uint

type UintNamedValue struct {
	Key string
	Val uint64
}

func (v *UintNamedValue) Name() string { return v.Key }

func (v *UintNamedValue) Log(message *Message) {
	message.Uint64(v.Key, v.Val)
}

type UintsNamedValue struct {
	Key  string
	Vals []uint64
}

func (v *UintsNamedValue) Name() string { return v.Key }

func (v *UintsNamedValue) Log(message *Message) {
	message.Uint64s(v.Key, v.Vals)
}

// Float

type FloatNamedValue struct {
	Key string
	Val float64
}

func (v *FloatNamedValue) Name() string { return v.Key }

func (v *FloatNamedValue) Log(message *Message) {
	message.Float(v.Key, v.Val)
}

type FloatsNamedValue struct {
	Key  string
	Vals []float64
}

func (v *FloatsNamedValue) Name() string { return v.Key }

func (v *FloatsNamedValue) Log(message *Message) {
	message.Floats(v.Key, v.Vals)
}

// String

type StringNamedValue struct {
	Key string
	Val string
}

func (v *StringNamedValue) Name() string { return v.Key }

func (v *StringNamedValue) Log(message *Message) {
	message.Str(v.Key, v.Val)
}

type StringsNamedValue struct {
	Key  string
	Vals []string
}

func (v *StringsNamedValue) Name() string { return v.Key }

func (v *StringsNamedValue) Log(message *Message) {
	message.Strs(v.Key, v.Vals)
}

// Error

type ErrorNamedValue struct {
	Key string
	Val error
}

func (v *ErrorNamedValue) Name() string { return v.Key }

func (v *ErrorNamedValue) Log(message *Message) {
	message.Error(v.Key, v.Val)
}

type ErrorsNamedValue struct {
	Key  string
	Vals []error
}

func (v *ErrorsNamedValue) Name() string { return v.Key }

func (v *ErrorsNamedValue) Log(message *Message) {
	message.Errors(v.Key, v.Vals)
}

// UUID

type UUIDNamedValue struct {
	Key string
	Val [16]byte
}

func (v *UUIDNamedValue) Name() string { return v.Key }

func (v *UUIDNamedValue) Log(message *Message) {
	message.UUID(v.Key, v.Val)
}

type UUIDsNamedValue struct {
	Key  string
	Vals [][16]byte
}

func (v *UUIDsNamedValue) Name() string { return v.Key }

func (v *UUIDsNamedValue) Log(message *Message) {
	message.UUIDs(v.Key, v.Vals)
}

// JSON

type JSONNamedValue struct {
	Key string
	Val []byte
}

func (v *JSONNamedValue) Name() string { return v.Key }

func (v *JSONNamedValue) Log(message *Message) {
	message.JSON(v.Key, v.Val)
}
