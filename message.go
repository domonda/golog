package golog

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"sync"
)

type Message struct {
	logger    *Logger
	level     Level
	formatter Formatter
}

var messagePool sync.Pool

func newMessage(logger *Logger, level Level, formatter Formatter) *Message {
	if m, ok := messagePool.Get().(*Message); ok {
		m.logger = logger
		m.level = level
		m.formatter = formatter
		return m
	}

	return &Message{
		logger:    logger,
		level:     level,
		formatter: formatter,
	}
}

func (m *Message) GetLevel() Level {
	return m.level
}

func (m *Message) IsActive() bool {
	return m != nil
}

func (m *Message) NewLogger() *Logger {
	if m == nil {
		return nil
	}
	return m.logger.WithFormatter(m.formatter.NewChild())
}

// Loggable lets a value that implements the Loggable log itself
func (m *Message) Loggable(key string, val Loggable) *Message {
	if m == nil {
		return nil
	}
	val.LogMessage(m, key)
	return m
}

// Err logs an error
func (m *Message) Err(key string, val error) *Message {
	if m == nil {
		return nil
	}
	m.formatter.WriteKey(key)
	m.formatter.WriteString(val.Error()) // TODO specialized WriteError ?
	return m
}

func (m *Message) Errs(key string, vals []error) *Message {
	if m == nil {
		return nil
	}
	m.formatter.WriteSliceKey(key)
	for _, val := range vals {
		m.formatter.WriteString(val.Error())
	}
	m.formatter.WriteSliceEnd()
	return m
}

func (m *Message) writeVal(key string, val interface{}) {
	m.formatter.WriteKey(key)

	if l, ok := val.(Loggable); ok {
		l.LogMessage(m, key)
	}

	v := reflect.ValueOf(val)
	for v.Kind() == reflect.Ptr && !v.IsNil() {
		v = v.Elem()
		val = v.Interface()
	}

	switch x := val.(type) {
	case Loggable:
		x.LogMessage(m, key)
	case bool:
		m.formatter.WriteBool(x)
	case int:
		m.formatter.WriteInt(int64(x))
	case int8:
		m.formatter.WriteInt(int64(x))
	case int16:
		m.formatter.WriteInt(int64(x))
	case int32:
		m.formatter.WriteInt(int64(x))
	case int64:
		m.formatter.WriteInt(x)
	case uint:
		m.formatter.WriteUint(uint64(x))
	case uint8:
		m.formatter.WriteUint(uint64(x))
	case uint16:
		m.formatter.WriteUint(uint64(x))
	case uint32:
		m.formatter.WriteUint(uint64(x))
	case uint64:
		m.formatter.WriteUint(x)
	case string:
		m.formatter.WriteString(x)
	case error:
		m.formatter.WriteString(x.Error())
	case nil:
		m.formatter.WriteString("<nil>") // TODO add a special WriteNil ?
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
	m.writeVal(key, val)
	return m
}

func (m *Message) Vals(key string, vals []interface{}) *Message {
	if m == nil {
		return nil
	}
	m.formatter.WriteSliceKey(key)
	for _, val := range vals {
		m.writeVal("", val) // TODO do we want empty string?
	}
	m.formatter.WriteSliceEnd()
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

func (m *Message) Bool(key string, val bool) *Message {
	if m == nil {
		return nil
	}
	m.formatter.WriteKey(key)
	m.formatter.WriteBool(val)
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

func (m *Message) UUID(key string, val [16]byte) *Message {
	if m == nil {
		return nil
	}
	m.formatter.WriteKey(key)
	m.formatter.WriteUUID(val)
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

	if json.Valid(val) {
		m.formatter.WriteKey(key)
		m.formatter.WriteJSON(val)
	}
	return m
}

func (m *Message) Log() {
	if m == nil {
		return
	}
	m.formatter.FlushAndFree()
	m.formatter = nil
	m.logger = nil
	messagePool.Put(m)
}

func (m *Message) LogAndExit() {
	m.Log()
	os.Exit(1)
}
