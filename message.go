package golog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"sync"

	"github.com/ungerik/go-reflection"
)

type Message struct {
	logger    *Logger
	formatter Formatter
	text      string
}

var messagePool sync.Pool

func newMessage(logger *Logger, formatter Formatter, text string) *Message {
	if m, ok := messagePool.Get().(*Message); ok {
		m.logger = logger
		m.formatter = formatter
		m.text = text
		return m
	}

	return &Message{
		logger:    logger,
		formatter: formatter,
		text:      text,
	}
}

func (m *Message) IsActive() bool {
	return m != nil
}

func (m *Message) NewLogger() *Logger {
	if m == nil {
		return nil
	}
	recorded, ok := m.formatter.(*recordingFormatter)
	if !ok {
		panic("golog.Message was not created by Logger.Record()")
	}
	return m.logger.WithHooks(recorded.hooks...)
}

// Loggable lets a value that implements the Loggable log itself
func (m *Message) Loggable(key string, val Loggable) *Message {
	if m == nil {
		return nil
	}
	val.LogMessage(m, key)
	return m
}

func (m *Message) Exec(writeFunc func(*Message)) *Message {
	if m == nil {
		return nil
	}
	writeFunc(m)
	return m
}

// Err is a shortcut for Error("error", val)
func (m *Message) Err(val error) *Message {
	return m.Error("error", val)
}

func (m *Message) Error(key string, val error) *Message {
	if m == nil {
		return nil
	}
	m.formatter.WriteKey(key)
	m.formatter.WriteError(val)
	return m
}

func (m *Message) Errors(key string, vals []error) *Message {
	if m == nil {
		return nil
	}
	m.formatter.WriteSliceKey(key)
	for _, val := range vals {
		m.formatter.WriteError(val)
	}
	m.formatter.WriteSliceEnd()
	return m
}

func (m *Message) writeVal(key string, val reflect.Value) {
	m.formatter.WriteKey(key)

	switch x := val.Interface().(type) {
	case nil:
		m.formatter.WriteNil()
		return
	case error:
		m.formatter.WriteError(x)
		return
	case Loggable:
		x.LogMessage(m, key)
		return
	}

	// Deref pointers
	for val.Kind() == reflect.Ptr && !val.IsNil() {
		val = val.Elem()
	}

	switch x := val.Interface().(type) {
	case nil:
		m.formatter.WriteNil()
		return
	case error:
		m.formatter.WriteError(x)
		return
	case Loggable:
		x.LogMessage(m, key)
		return
	}

	switch val.Kind() {
	case reflect.Bool:
		m.formatter.WriteBool(val.Bool())
	case reflect.Int:
		m.formatter.WriteInt(val.Int())
	case reflect.Int8:
		m.formatter.WriteInt(val.Int())
	case reflect.Int16:
		m.formatter.WriteInt(val.Int())
	case reflect.Int32:
		m.formatter.WriteInt(val.Int())
	case reflect.Int64:
		m.formatter.WriteInt(val.Int())
	case reflect.Uint:
		m.formatter.WriteUint(val.Uint())
	case reflect.Uint8:
		m.formatter.WriteUint(val.Uint())
	case reflect.Uint16:
		m.formatter.WriteUint(val.Uint())
	case reflect.Uint32:
		m.formatter.WriteUint(val.Uint())
	case reflect.Uint64:
		m.formatter.WriteUint(val.Uint())
	case reflect.String:
		m.formatter.WriteString(val.String())
	default:
		m.formatter.WriteString(fmt.Sprint(val))
	}
}

// Val logs val with the best matching typed log method
// or uses Print if none was found.
func (m *Message) Val(key string, val interface{}) *Message {
	if m == nil {
		return nil
	}
	m.writeVal(key, reflect.ValueOf(val))
	return m
}

func (m *Message) Vals(key string, vals []interface{}) *Message {
	if m == nil {
		return nil
	}
	m.formatter.WriteSliceKey(key)
	for _, val := range vals {
		m.writeVal("", reflect.ValueOf(val)) // TODO do we want empty string?
	}
	m.formatter.WriteSliceEnd()
	return m
}

// StructFields calls Val(fieldName, fieldValue) for every exported struct field
func (m *Message) StructFields(strct interface{}) *Message {
	if m == nil {
		return nil
	}
	reflection.EnumFlatExportedStructFields(strct, func(field reflect.StructField, value reflect.Value) {
		m.writeVal(field.Name, value)
	})
	return m
}

// Print logs vals as string with the "%v" format of the fmt package.
// If only one value is passed for vals, then it will be logged as single string,
// else a slice of strings will be logged for vals.
func (m *Message) Print(key string, vals ...interface{}) *Message {
	if m == nil {
		return nil
	}
	if len(vals) == 1 {
		m.formatter.WriteKey(key)
		m.formatter.WriteString(fmt.Sprint(vals...))
	} else {
		m.formatter.WriteSliceKey(key)
		for _, val := range vals {
			m.formatter.WriteString(fmt.Sprint(val))
		}
		m.formatter.WriteSliceEnd()
	}
	return m
}

func (m *Message) Nil(key string) *Message {
	if m == nil {
		return nil
	}
	m.formatter.WriteKey(key)
	m.formatter.WriteNil()
	return m
}

func (m *Message) Bool(key string, val bool) *Message {
	if m == nil {
		return nil
	}
	m.formatter.WriteKey(key)
	m.formatter.WriteBool(val)
	return m
}

func (m *Message) BoolPtr(key string, val *bool) *Message {
	if m == nil {
		return nil
	}
	m.formatter.WriteKey(key)
	if val != nil {
		m.formatter.WriteBool(*val)
	} else {
		m.formatter.WriteNil()
	}
	return m
}

func (m *Message) Bools(key string, vals []bool) *Message {
	if m == nil {
		return nil
	}
	m.formatter.WriteSliceKey(key)
	for _, val := range vals {
		m.formatter.WriteBool(val)
	}
	m.formatter.WriteSliceEnd()
	return m
}

func (m *Message) Int(key string, val int) *Message {
	if m == nil {
		return nil
	}
	m.formatter.WriteKey(key)
	m.formatter.WriteInt(int64(val))
	return m
}

func (m *Message) IntPtr(key string, val *int) *Message {
	if m == nil {
		return nil
	}
	m.formatter.WriteKey(key)
	if val != nil {
		m.formatter.WriteInt(int64(*val))
	} else {
		m.formatter.WriteNil()
	}
	return m
}

func (m *Message) Ints(key string, vals []int) *Message {
	if m == nil {
		return nil
	}
	m.formatter.WriteSliceKey(key)
	for _, val := range vals {
		m.formatter.WriteInt(int64(val))
	}
	m.formatter.WriteSliceEnd()
	return m
}

func (m *Message) Int8(key string, val int8) *Message {
	if m == nil {
		return nil
	}
	m.formatter.WriteKey(key)
	m.formatter.WriteInt(int64(val))
	return m
}

func (m *Message) Int8Ptr(key string, val *int8) *Message {
	if m == nil {
		return nil
	}
	m.formatter.WriteKey(key)
	if val != nil {
		m.formatter.WriteInt(int64(*val))
	} else {
		m.formatter.WriteNil()
	}
	return m
}

func (m *Message) Int8s(key string, vals []int8) *Message {
	if m == nil {
		return nil
	}
	m.formatter.WriteSliceKey(key)
	for _, val := range vals {
		m.formatter.WriteInt(int64(val))
	}
	m.formatter.WriteSliceEnd()
	return m
}

func (m *Message) Int16(key string, val int16) *Message {
	if m == nil {
		return nil
	}
	m.formatter.WriteKey(key)
	m.formatter.WriteInt(int64(val))
	return m
}

func (m *Message) Int16Ptr(key string, val *int16) *Message {
	if m == nil {
		return nil
	}
	m.formatter.WriteKey(key)
	if val != nil {
		m.formatter.WriteInt(int64(*val))
	} else {
		m.formatter.WriteNil()
	}
	return m
}

func (m *Message) Int16s(key string, vals []int16) *Message {
	if m == nil {
		return nil
	}
	m.formatter.WriteSliceKey(key)
	for _, val := range vals {
		m.formatter.WriteInt(int64(val))
	}
	m.formatter.WriteSliceEnd()
	return m
}

func (m *Message) Int32(key string, val int32) *Message {
	if m == nil {
		return nil
	}
	m.formatter.WriteKey(key)
	m.formatter.WriteInt(int64(val))
	return m
}

func (m *Message) Int32Ptr(key string, val *int32) *Message {
	if m == nil {
		return nil
	}
	m.formatter.WriteKey(key)
	if val != nil {
		m.formatter.WriteInt(int64(*val))
	} else {
		m.formatter.WriteNil()
	}
	return m
}

func (m *Message) Int32s(key string, vals []int32) *Message {
	if m == nil {
		return nil
	}
	m.formatter.WriteSliceKey(key)
	for _, val := range vals {
		m.formatter.WriteInt(int64(val))
	}
	m.formatter.WriteSliceEnd()
	return m
}

func (m *Message) Int64(key string, val int64) *Message {
	if m == nil {
		return nil
	}
	m.formatter.WriteKey(key)
	m.formatter.WriteInt(val)
	return m
}

func (m *Message) Int64Ptr(key string, val *int64) *Message {
	if m == nil {
		return nil
	}
	m.formatter.WriteKey(key)
	if val != nil {
		m.formatter.WriteInt(*val)
	} else {
		m.formatter.WriteNil()
	}
	return m
}

func (m *Message) Int64s(key string, vals []int64) *Message {
	if m == nil {
		return nil
	}
	m.formatter.WriteSliceKey(key)
	for _, val := range vals {
		m.formatter.WriteInt(val)
	}
	m.formatter.WriteSliceEnd()
	return m
}

func (m *Message) Uint(key string, val uint) *Message {
	if m == nil {
		return nil
	}
	m.formatter.WriteKey(key)
	m.formatter.WriteUint(uint64(val))
	return m
}

func (m *Message) UintPtr(key string, val *uint) *Message {
	if m == nil {
		return nil
	}
	m.formatter.WriteKey(key)
	if val != nil {
		m.formatter.WriteUint(uint64(*val))
	} else {
		m.formatter.WriteNil()
	}
	return m
}

func (m *Message) Uints(key string, vals []uint) *Message {
	if m == nil {
		return nil
	}
	m.formatter.WriteSliceKey(key)
	for _, val := range vals {
		m.formatter.WriteUint(uint64(val))
	}
	m.formatter.WriteSliceEnd()
	return m
}

func (m *Message) Uint8(key string, val uint8) *Message {
	if m == nil {
		return nil
	}
	m.formatter.WriteKey(key)
	m.formatter.WriteUint(uint64(val))
	return m
}

func (m *Message) Uint8Ptr(key string, val *uint8) *Message {
	if m == nil {
		return nil
	}
	m.formatter.WriteKey(key)
	if val != nil {
		m.formatter.WriteUint(uint64(*val))
	} else {
		m.formatter.WriteNil()
	}
	return m
}

func (m *Message) Uint8s(key string, vals []uint8) *Message {
	if m == nil {
		return nil
	}
	m.formatter.WriteSliceKey(key)
	for _, val := range vals {
		m.formatter.WriteUint(uint64(val))
	}
	m.formatter.WriteSliceEnd()
	return m
}

func (m *Message) Uint16(key string, val uint16) *Message {
	if m == nil {
		return nil
	}
	m.formatter.WriteKey(key)
	m.formatter.WriteUint(uint64(val))
	return m
}

func (m *Message) Uint16Ptr(key string, val *uint16) *Message {
	if m == nil {
		return nil
	}
	m.formatter.WriteKey(key)
	if val != nil {
		m.formatter.WriteUint(uint64(*val))
	} else {
		m.formatter.WriteNil()
	}
	return m
}

func (m *Message) Uint16s(key string, vals []uint16) *Message {
	if m == nil {
		return nil
	}
	m.formatter.WriteSliceKey(key)
	for _, val := range vals {
		m.formatter.WriteUint(uint64(val))
	}
	m.formatter.WriteSliceEnd()
	return m
}

func (m *Message) Uint32(key string, val uint32) *Message {
	if m == nil {
		return nil
	}
	m.formatter.WriteKey(key)
	m.formatter.WriteUint(uint64(val))
	return m
}

func (m *Message) Uint32Ptr(key string, val *uint32) *Message {
	if m == nil {
		return nil
	}
	m.formatter.WriteKey(key)
	if val != nil {
		m.formatter.WriteUint(uint64(*val))
	} else {
		m.formatter.WriteNil()
	}
	return m
}

func (m *Message) Uint32s(key string, vals []uint32) *Message {
	if m == nil {
		return nil
	}
	m.formatter.WriteSliceKey(key)
	for _, val := range vals {
		m.formatter.WriteUint(uint64(val))
	}
	m.formatter.WriteSliceEnd()
	return m
}

func (m *Message) Uint64(key string, val uint64) *Message {
	if m == nil {
		return nil
	}
	m.formatter.WriteKey(key)
	m.formatter.WriteUint(val)
	return m
}

func (m *Message) Uint64Ptr(key string, val *uint64) *Message {
	if m == nil {
		return nil
	}
	m.formatter.WriteKey(key)
	if val != nil {
		m.formatter.WriteUint(*val)
	} else {
		m.formatter.WriteNil()
	}
	return m
}

func (m *Message) Uint64s(key string, vals []uint64) *Message {
	if m == nil {
		return nil
	}
	m.formatter.WriteSliceKey(key)
	for _, val := range vals {
		m.formatter.WriteUint(val)
	}
	m.formatter.WriteSliceEnd()
	return m
}

func (m *Message) Float32(key string, val float32) *Message {
	if m == nil {
		return nil
	}
	m.formatter.WriteKey(key)
	m.formatter.WriteFloat(float64(val))
	return m
}

func (m *Message) Float32Ptr(key string, val *float32) *Message {
	if m == nil {
		return nil
	}
	m.formatter.WriteKey(key)
	if val != nil {
		m.formatter.WriteFloat(float64(*val))
	} else {
		m.formatter.WriteNil()
	}
	return m
}

func (m *Message) Float32s(key string, vals []float32) *Message {
	if m == nil {
		return nil
	}
	m.formatter.WriteSliceKey(key)
	for _, val := range vals {
		m.formatter.WriteFloat(float64(val))
	}
	m.formatter.WriteSliceEnd()
	return m
}

// Float is not called Float64 on purpose
func (m *Message) Float(key string, val float64) *Message {
	if m == nil {
		return nil
	}
	m.formatter.WriteKey(key)
	m.formatter.WriteFloat(val)
	return m
}

func (m *Message) FloatPtr(key string, val *float64) *Message {
	if m == nil {
		return nil
	}
	m.formatter.WriteKey(key)
	if val != nil {
		m.formatter.WriteFloat(*val)
	} else {
		m.formatter.WriteNil()
	}
	return m
}

func (m *Message) Floats(key string, vals []float64) *Message {
	if m == nil {
		return nil
	}
	m.formatter.WriteSliceKey(key)
	for _, val := range vals {
		m.formatter.WriteFloat(val)
	}
	m.formatter.WriteSliceEnd()
	return m
}

func (m *Message) Str(key, val string) *Message {
	if m == nil {
		return nil
	}
	m.formatter.WriteKey(key)
	m.formatter.WriteString(val)
	return m
}

func (m *Message) StrPtr(key string, val *string) *Message {
	if m == nil {
		return nil
	}
	m.formatter.WriteKey(key)
	if val != nil {
		m.formatter.WriteString(*val)
	} else {
		m.formatter.WriteNil()
	}
	return m
}

func (m *Message) Strs(key string, vals []string) *Message {
	if m == nil {
		return nil
	}
	m.formatter.WriteSliceKey(key)
	for _, val := range vals {
		m.formatter.WriteString(val)
	}
	m.formatter.WriteSliceEnd()
	return m
}

func (m *Message) Stringer(key string, val fmt.Stringer) *Message {
	return m.Str(key, val.String())
}

func (m *Message) UUID(key string, val [16]byte) *Message {
	if m == nil {
		return nil
	}
	m.formatter.WriteKey(key)
	m.formatter.WriteUUID(val)
	return m
}

func (m *Message) UUIDPtr(key string, val *[16]byte) *Message {
	if m == nil {
		return nil
	}
	m.formatter.WriteKey(key)
	if val != nil {
		m.formatter.WriteUUID(*val)
	} else {
		m.formatter.WriteNil()
	}
	return m
}

func (m *Message) UUIDs(key string, vals [][16]byte) *Message {
	if m == nil {
		return nil
	}
	m.formatter.WriteSliceKey(key)
	for _, val := range vals {
		m.formatter.WriteUUID(val)
	}
	m.formatter.WriteSliceEnd()
	return m
}

func (m *Message) JSON(key string, val []byte) *Message {
	if m == nil {
		return nil
	}
	buf := bytes.NewBuffer(make([]byte, 0, len(val)))
	err := json.Compact(buf, val)
	m.formatter.WriteKey(key)
	if err == nil {
		m.formatter.WriteJSON(buf.Bytes())
	} else {
		m.formatter.WriteJSON(nil)
	}
	return m
}

// Log writes the complete log message
// and returns the Message to a sync.Pool.
func (m *Message) Log() {
	if m == nil {
		return
	}
	m.formatter.FlushAndFree()
	m.formatter = nil
	m.logger = nil
	m.text = ""
	messagePool.Put(m)
}

// LogAndPanic writes the complete log message
// and panics with the message text.
func (m *Message) LogAndPanic() {
	m.Log()
	panic(m.text)
}
