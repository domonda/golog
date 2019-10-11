package golog

type Hook interface {
	Log(*Message)
}

type HookFunc func(*Message)

func (f HookFunc) Log(message *Message) { f(message) }

// Nil

type NilHook struct {
	Key string
}

func (h *NilHook) Log(message *Message) {
	message.Nil(h.Key)
}

// Bool

type BoolHook struct {
	Key string
	Val bool
}

func (h *BoolHook) Log(message *Message) {
	message.Bool(h.Key, h.Val)
}

type BoolsHook struct {
	Key  string
	Vals []bool
}

func (h *BoolsHook) Log(message *Message) {
	message.Bools(h.Key, h.Vals)
}

// Int

type IntHook struct {
	Key string
	Val int64
}

func (h *IntHook) Log(message *Message) {
	message.Int64(h.Key, h.Val)
}

type IntsHook struct {
	Key  string
	Vals []int64
}

func (h *IntsHook) Log(message *Message) {
	message.Int64s(h.Key, h.Vals)
}

// Uint

type UintHook struct {
	Key string
	Val uint64
}

func (h *UintHook) Log(message *Message) {
	message.Uint64(h.Key, h.Val)
}

type UintsHook struct {
	Key  string
	Vals []uint64
}

func (h *UintsHook) Log(message *Message) {
	message.Uint64s(h.Key, h.Vals)
}

// Float

type FloatHook struct {
	Key string
	Val float64
}

func (h *FloatHook) Log(message *Message) {
	message.Float(h.Key, h.Val)
}

type FloatsHook struct {
	Key  string
	Vals []float64
}

func (h *FloatsHook) Log(message *Message) {
	message.Floats(h.Key, h.Vals)
}

// String

type StringHook struct {
	Key string
	Val string
}

func (h *StringHook) Log(message *Message) {
	message.Str(h.Key, h.Val)
}

type StringsHook struct {
	Key  string
	Vals []string
}

func (h *StringsHook) Log(message *Message) {
	message.Strs(h.Key, h.Vals)
}

// Error

type ErrorHook struct {
	Key string
	Val error
}

func (h *ErrorHook) Log(message *Message) {
	message.Err(h.Key, h.Val)
}

type ErrorsHook struct {
	Key  string
	Vals []error
}

func (h *ErrorsHook) Log(message *Message) {
	message.Errs(h.Key, h.Vals)
}

// UUID

type UUIDHook struct {
	Key string
	Val [16]byte
}

func (h *UUIDHook) Log(message *Message) {
	message.UUID(h.Key, h.Val)
}

type UUIDsHook struct {
	Key  string
	Vals [][16]byte
}

func (h *UUIDsHook) Log(message *Message) {
	message.UUIDs(h.Key, h.Vals)
}

// JSON

type JSONHook struct {
	Key string
	Val []byte
}

func (h *JSONHook) Log(message *Message) {
	message.JSON(h.Key, h.Val)
}
