package golog

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"go/token"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

type Message struct {
	logger  *Logger
	attribs Attribs
	writers []Writer
	level   Level
	text    string // Used for LogAndPanic
}

var (
	messagePool sync.Pool
	writersPool sync.Pool
)

func messageFromPool(logger *Logger, attribs Attribs, writers []Writer, level Level, text string) *Message {
	if m, ok := messagePool.Get().(*Message); ok {
		m.logger = logger
		m.attribs = attribs
		m.writers = writers
		m.level = level
		m.text = text
		return m
	}
	return &Message{
		logger:  logger,
		attribs: attribs,
		writers: writers,
		level:   level,
		text:    text,
	}
}

// IsActive returns true if the message is not nil.
func (m *Message) IsActive() bool {
	return m != nil
}

// IsAttribRecorder returns true if the Message was created by Logger.With()
// for recording attribs instead of immediately logging them.
func (m *Message) IsAttribRecorder() bool {
	return m != nil && len(m.writers) == 0
}

// SubLogger returns a new sub-logger with recorded per message attribs.
func (m *Message) SubLogger() *Logger {
	if m == nil {
		return nil
	}
	if !m.IsAttribRecorder() {
		// Message was not created by Logger.With() for recording attribs
		return m.logger
	}
	return m.logger.WithAttribs(m.attribs...)
}

// SubLoggerContext returns a new sub-logger with recorded per message attribs
// in addition to any attribs from the passed ctx
// and a context with those attribs added to it.
func (m *Message) SubLoggerContext(ctx context.Context) (*Logger, context.Context) {
	if m == nil {
		return nil, ctx
	}
	logger := m.logger.WithAdditionalWriterConfigs(
		WriterConfigsFromContext(ctx)...,
	)
	if m.IsAttribRecorder() {
		// Message was created by Logger.With()
		// for recording attribs so use those recorded attribs
		attribs := m.attribs.AppendUnique(AttribsFromContext(ctx)...)
		logger = logger.WithAttribs(attribs...)
	}
	return logger, m.attribs.AddToContext(ctx)
}

// SubContext returns a new context with recorded per message attribs
// added to the passed ctx argument.
func (m *Message) SubContext(ctx context.Context) context.Context {
	if m == nil {
		return ctx
	}
	return m.attribs.AddToContext(ctx)
}

// Ctx logs any attribs that were added to the context
// and that are not already in the logger's attribs.
func (m *Message) Ctx(ctx context.Context) *Message {
	if m == nil {
		return nil
	}
	AttribsFromContext(ctx).Log(m)
	return m
}

// Loggable lets an implementation of the Loggable interface log itself
func (m *Message) Loggable(loggable Loggable) *Message {
	if m == nil || loggable == nil {
		return m
	}
	loggable.Log(m)
	return m
}

func (m *Message) Exec(logFunc func(*Message)) *Message {
	if m == nil || logFunc == nil {
		return m
	}
	logFunc(m)
	return m
}

// Err is a shortcut for Error("error", val)
func (m *Message) Err(val error) *Message {
	return m.Error("error", val)
}

func (m *Message) Error(key string, val error) *Message {
	if m == nil || m.attribs.Has(key) {
		return m
	}
	if m.IsAttribRecorder() {
		m.attribs = append(m.attribs, Error{Key: key, Val: val})
		return m
	}
	if val == nil {
		return m.Nil(key)
	}
	for _, w := range m.writers {
		w.WriteKey(key)
		w.WriteError(val)
	}
	return m
}

// Errs is a shortcut for Errors("errors", vals)
func (m *Message) Errs(vals []error) *Message {
	return m.Errors("errors", vals)
}

func (m *Message) Errors(key string, vals []error) *Message {
	if m == nil || m.attribs.Has(key) {
		return m
	}
	if m.IsAttribRecorder() {
		m.attribs = append(m.attribs, Errors{Key: key, Vals: vals})
		return m
	}
	for _, w := range m.writers {
		w.WriteSliceKey(key)
		for _, val := range vals {
			w.WriteError(val)
		}
		w.WriteSliceEnd()
	}
	return m
}

// CallStack logs the current call stack
// as an error value with the passed key.
func (m *Message) CallStack(key string) *Message {
	return m.CallStackSkip(key, 1)
}

// CallStack logs the current call stack
// as an error value with the passed key,
// with skip number of top frames omitted.
func (m *Message) CallStackSkip(key string, skip int) *Message {
	return m.Error(key, errors.New(callstack(1+skip)))
}

// Any logs val with the best matching typed log method
// or uses Print if none was found.
func (m *Message) Any(key string, val any) *Message {
	if m == nil || m.attribs.Has(key) {
		return m
	}
	if m.IsAttribRecorder() {
		m.attribs = append(m.attribs, Any{Key: key, Val: val})
		return m
	}

	if val == nil {
		return m.Nil(key)
	}

	v := reflect.ValueOf(val)

	if isSlice(v) {
		for _, w := range m.writers {
			w.WriteSliceKey(key)
			m.writeAny(w, v, false)
			w.WriteSliceEnd()
		}
		return m
	}

	for _, w := range m.writers {
		w.WriteKey(key)
		m.writeAny(w, v, false)
	}
	return m
}

func isSlice(v reflect.Value) bool {
	if !v.IsValid() {
		return false
	}
	for v.Kind() == reflect.Pointer && !v.IsNil() {
		v = v.Elem()
	}
	return v.Kind() == reflect.Slice || (v.Kind() == reflect.Array && !isUUID(v))
}

func (m *Message) writeAny(w Writer, val reflect.Value, nestedSlice bool) {
	// Try if val implements a loggable interface or is nil
	written := m.tryWriteInterface(w, val)
	if written {
		return
	}

	// Deref pointers
	dereferenced := false
	for val.Kind() == reflect.Pointer && !val.IsNil() {
		val = val.Elem()
		dereferenced = true
	}
	if dereferenced {
		// Try if dereferenced val implements a loggable interface or is nil
		written := m.tryWriteInterface(w, val)
		if written {
			return
		}
	}

	switch val.Kind() {
	case reflect.Pointer:
		// A non-nil pointer would have been dereferenced above
		w.WriteNil()

	case reflect.Bool:
		w.WriteBool(val.Bool())

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		w.WriteInt(val.Int())

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		w.WriteUint(val.Uint())

	case reflect.Float32, reflect.Float64:
		w.WriteFloat(val.Float())

	case reflect.String:
		w.WriteString(val.String())

	case reflect.Struct, reflect.Map:
		j, err := json.Marshal(val.Interface())
		if err != nil {
			w.WriteError(fmt.Errorf("error while marshalling %s as JSON for logging: %w", val.Type(), err))
			return
		}
		w.WriteJSON(j)

	case reflect.Array:
		if uuid, ok := asUUID(val); ok {
			if IsNilUUID(uuid) {
				w.WriteNil()
			} else {
				w.WriteUUID(uuid)
			}
			return
		}
		if nestedSlice {
			// Don't go further into a slice of slices
			w.WriteString(fmt.Sprint(val))
		} else {
			for i := 0; i < val.Len(); i++ {
				m.writeAny(w,
					val.Index(i),
					true, // nestedSlice
				)
			}
		}

	case reflect.Slice:
		if nestedSlice {
			// Don't go further into a slice of slices
			w.WriteString(fmt.Sprint(val))
		} else {
			for i := 0; i < val.Len(); i++ {
				m.writeAny(w,
					val.Index(i),
					true, // nestedSlice
				)
			}
		}

	default:
		w.WriteString(fmt.Sprint(val))
	}
}

func (m *Message) tryWriteInterface(w Writer, val reflect.Value) (written bool) {
	if nullable, ok := val.Interface().(interface{ IsNull() bool }); ok && nullable.IsNull() {
		w.WriteNil()
		return true
	}

	switch x := val.Interface().(type) {
	case nil:
		w.WriteNil()
		return true

	case Loggable:
		x.Log(m)
		return true

	case error:
		w.WriteError(x)
		return true

	case [16]byte:
		if IsNilUUID(x) {
			w.WriteNil()
		} else {
			w.WriteUUID(x)
		}
		return true
	}

	return false
}

// StructFields calls Any(fieldName, fieldValue) for every exported struct field
func (m *Message) StructFields(strct any) *Message {
	if m == nil || strct == nil {
		return m
	}
	m.structFields(reflect.ValueOf(strct), "")
	return m
}

// TaggedStructFields calls Any(fieldTag, fieldValue) for every exported struct field
// that has the passed tag with the tag value not being empty or "-".
// Tag values are only considered until the first comma character,
// so `tag:"hello_world,omitempty"` will result in the fieldTag "hello_world".
// Fields with the following tags will be ignored: `tag:"-"`, `tag:""` `tag:",xxx"`.
func (m *Message) TaggedStructFields(strct any, tag string) *Message {
	if m == nil || strct == nil {
		return m
	}
	m.structFields(reflect.ValueOf(strct), tag)
	return m
}

func (m *Message) structFields(v reflect.Value, tag string) {
	for v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return
		}
		v = v.Elem()
	}
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		switch {
		case field.Anonymous:
			m.structFields(v.Field(i), tag)
		case token.IsExported(field.Name):
			name := field.Name
			if tag != "" {
				n := field.Tag.Get(tag)
				if c := strings.IndexByte(n, ','); c >= 0 {
					n = n[:c]
				}
				if n == "" || n == "-" {
					continue
				}
				name = n
			}
			m.Any(name, v.Field(i).Interface())
		}
	}
}

// Print logs vals as string with the "%v" format of the fmt package.
// If only one value is passed for vals, then it will be logged as single string,
// else a slice of strings will be logged for vals.
func (m *Message) Print(key string, vals ...any) *Message {
	if m == nil || m.attribs.Has(key) {
		return m
	}
	if m.IsAttribRecorder() {
		if len(vals) == 1 {
			m.attribs = append(m.attribs, String{Key: key, Val: fmt.Sprint(vals...)})
		} else {
			strs := make([]string, len(vals))
			for i, val := range vals {
				strs[i] = fmt.Sprint(val)
			}
			m.attribs = append(m.attribs, Strings{Key: key, Vals: strs})
		}
		return m
	}
	if len(vals) == 1 {
		for _, w := range m.writers {
			w.WriteKey(key)
			w.WriteString(fmt.Sprint(vals...))
		}
	} else {
		for _, w := range m.writers {
			w.WriteSliceKey(key)
			for _, val := range vals {
				w.WriteString(fmt.Sprint(val))
			}
			w.WriteSliceEnd()
		}
	}
	return m
}

func (m *Message) Nil(key string) *Message {
	if m == nil || m.attribs.Has(key) {
		return m
	}
	if m.IsAttribRecorder() {
		m.attribs = append(m.attribs, Nil{Key: key})
		return m
	}
	for _, w := range m.writers {
		w.WriteKey(key)
		w.WriteNil()
	}
	return m
}

func (m *Message) Bool(key string, val bool) *Message {
	if m == nil || m.attribs.Has(key) {
		return m
	}
	if m.IsAttribRecorder() {
		m.attribs = append(m.attribs, Bool{Key: key, Val: val})
		return m
	}
	for _, w := range m.writers {
		w.WriteKey(key)
		w.WriteBool(val)
	}
	return m
}

func (m *Message) BoolPtr(key string, val *bool) *Message {
	if val == nil {
		return m.Nil(key)
	}
	return m.Bool(key, *val)
}

func (m *Message) Bools(key string, vals []bool) *Message {
	if m == nil || m.attribs.Has(key) {
		return m
	}
	if m.IsAttribRecorder() {
		valsCpy := append([]bool(nil), vals...)
		m.attribs = append(m.attribs, Bools{Key: key, Vals: valsCpy})
		return m
	}
	for _, w := range m.writers {
		w.WriteSliceKey(key)
		for _, val := range vals {
			w.WriteBool(val)
		}
		w.WriteSliceEnd()
	}
	return m
}

func (m *Message) Int(key string, val int) *Message {
	if m == nil || m.attribs.Has(key) {
		return m
	}
	if m.IsAttribRecorder() {
		m.attribs = append(m.attribs, Int{Key: key, Val: int64(val)})
		return m
	}
	for _, w := range m.writers {
		w.WriteKey(key)
		w.WriteInt(int64(val))
	}
	return m
}

func (m *Message) IntPtr(key string, val *int) *Message {
	if val == nil {
		return m.Nil(key)
	}
	return m.Int(key, *val)
}

func (m *Message) Ints(key string, vals []int) *Message {
	if m == nil || m.attribs.Has(key) {
		return m
	}
	if m.IsAttribRecorder() {
		valsCpy := make([]int64, len(vals))
		for i, val := range vals {
			valsCpy[i] = int64(val)
		}
		m.attribs = append(m.attribs, Ints{Key: key, Vals: valsCpy})
		return m
	}
	for _, w := range m.writers {
		w.WriteSliceKey(key)
		for _, val := range vals {
			w.WriteInt(int64(val))
		}
		w.WriteSliceEnd()
	}
	return m
}

func (m *Message) Int8(key string, val int8) *Message {
	if m == nil || m.attribs.Has(key) {
		return m
	}
	if m.IsAttribRecorder() {
		m.attribs = append(m.attribs, Int{Key: key, Val: int64(val)})
		return m
	}
	for _, w := range m.writers {
		w.WriteKey(key)
		w.WriteInt(int64(val))
	}
	return m
}

func (m *Message) Int8Ptr(key string, val *int8) *Message {
	if val == nil {
		return m.Nil(key)
	}
	return m.Int8(key, *val)
}

func (m *Message) Int8s(key string, vals []int8) *Message {
	if m == nil || m.attribs.Has(key) {
		return m
	}
	if m.IsAttribRecorder() {
		valsCpy := make([]int64, len(vals))
		for i, val := range vals {
			valsCpy[i] = int64(val)
		}
		m.attribs = append(m.attribs, Ints{Key: key, Vals: valsCpy})
		return m
	}
	for _, w := range m.writers {
		w.WriteSliceKey(key)
		for _, val := range vals {
			w.WriteInt(int64(val))
		}
		w.WriteSliceEnd()
	}
	return m
}

func (m *Message) Int16(key string, val int16) *Message {
	if m == nil || m.attribs.Has(key) {
		return m
	}
	if m.IsAttribRecorder() {
		m.attribs = append(m.attribs, Int{Key: key, Val: int64(val)})
		return m
	}
	for _, w := range m.writers {
		w.WriteKey(key)
		w.WriteInt(int64(val))
	}
	return m
}

func (m *Message) Int16Ptr(key string, val *int16) *Message {
	if val == nil {
		return m.Nil(key)
	}
	return m.Int16(key, *val)
}

func (m *Message) Int16s(key string, vals []int16) *Message {
	if m == nil || m.attribs.Has(key) {
		return m
	}
	if m.IsAttribRecorder() {
		valsCpy := make([]int64, len(vals))
		for i, val := range vals {
			valsCpy[i] = int64(val)
		}
		m.attribs = append(m.attribs, Ints{Key: key, Vals: valsCpy})
		return m
	}
	for _, w := range m.writers {
		w.WriteSliceKey(key)
		for _, val := range vals {
			w.WriteInt(int64(val))
		}
		w.WriteSliceEnd()
	}
	return m
}

func (m *Message) Int32(key string, val int32) *Message {
	if m == nil || m.attribs.Has(key) {
		return m
	}
	if m.IsAttribRecorder() {
		m.attribs = append(m.attribs, Int{Key: key, Val: int64(val)})
		return m
	}
	for _, w := range m.writers {
		w.WriteKey(key)
		w.WriteInt(int64(val))
	}
	return m
}

func (m *Message) Int32Ptr(key string, val *int32) *Message {
	if val == nil {
		return m.Nil(key)
	}
	return m.Int32(key, *val)
}

func (m *Message) Int32s(key string, vals []int32) *Message {
	if m == nil || m.attribs.Has(key) {
		return m
	}
	if m.IsAttribRecorder() {
		valsCpy := make([]int64, len(vals))
		for i, val := range vals {
			valsCpy[i] = int64(val)
		}
		m.attribs = append(m.attribs, Ints{Key: key, Vals: valsCpy})
		return m
	}
	for _, w := range m.writers {
		w.WriteSliceKey(key)
		for _, val := range vals {
			w.WriteInt(int64(val))
		}
		w.WriteSliceEnd()
	}
	return m
}

func (m *Message) Int64(key string, val int64) *Message {
	if m == nil || m.attribs.Has(key) {
		return m
	}
	if m.IsAttribRecorder() {
		m.attribs = append(m.attribs, Int{Key: key, Val: val})
		return m
	}
	for _, w := range m.writers {
		w.WriteKey(key)
		w.WriteInt(val)
	}
	return m
}

func (m *Message) Int64Ptr(key string, val *int64) *Message {
	if val == nil {
		return m.Nil(key)
	}
	return m.Int64(key, *val)
}

func (m *Message) Int64s(key string, vals []int64) *Message {
	if m == nil || m.attribs.Has(key) {
		return m
	}
	if m.IsAttribRecorder() {
		valsCpy := append([]int64(nil), vals...)
		m.attribs = append(m.attribs, Ints{Key: key, Vals: valsCpy})
		return m
	}
	for _, w := range m.writers {
		w.WriteSliceKey(key)
		for _, val := range vals {
			w.WriteInt(val)
		}
		w.WriteSliceEnd()
	}
	return m
}

func (m *Message) Uint(key string, val uint) *Message {
	if m == nil || m.attribs.Has(key) {
		return m
	}
	if m.IsAttribRecorder() {
		m.attribs = append(m.attribs, Uint{Key: key, Val: uint64(val)})
		return m
	}
	for _, w := range m.writers {
		w.WriteKey(key)
		w.WriteUint(uint64(val))
	}
	return m
}

func (m *Message) UintPtr(key string, val *uint) *Message {
	if val == nil {
		return m.Nil(key)
	}
	return m.Uint(key, *val)
}

func (m *Message) Uints(key string, vals []uint) *Message {
	if m == nil || m.attribs.Has(key) {
		return m
	}
	if m.IsAttribRecorder() {
		valsCpy := make([]uint64, len(vals))
		for i, val := range vals {
			valsCpy[i] = uint64(val)
		}
		m.attribs = append(m.attribs, Uints{Key: key, Vals: valsCpy})
		return m
	}
	for _, w := range m.writers {
		w.WriteSliceKey(key)
		for _, val := range vals {
			w.WriteUint(uint64(val))
		}
		w.WriteSliceEnd()
	}
	return m
}

func (m *Message) Uint8(key string, val uint8) *Message {
	if m == nil || m.attribs.Has(key) {
		return m
	}
	if m.IsAttribRecorder() {
		m.attribs = append(m.attribs, Uint{Key: key, Val: uint64(val)})
		return m
	}
	for _, w := range m.writers {
		w.WriteKey(key)
		w.WriteUint(uint64(val))
	}
	return m
}

func (m *Message) Uint8Ptr(key string, val *uint8) *Message {
	if val == nil {
		return m.Nil(key)
	}
	return m.Uint8(key, *val)
}

func (m *Message) Uint8s(key string, vals []uint8) *Message {
	if m == nil || m.attribs.Has(key) {
		return m
	}
	if m.IsAttribRecorder() {
		valsCpy := make([]uint64, len(vals))
		for i, val := range vals {
			valsCpy[i] = uint64(val)
		}
		m.attribs = append(m.attribs, Uints{Key: key, Vals: valsCpy})
		return m
	}
	for _, w := range m.writers {
		w.WriteSliceKey(key)
		for _, val := range vals {
			w.WriteUint(uint64(val))
		}
		w.WriteSliceEnd()
	}
	return m
}

func (m *Message) Uint16(key string, val uint16) *Message {
	if m == nil || m.attribs.Has(key) {
		return m
	}
	if m.IsAttribRecorder() {
		m.attribs = append(m.attribs, Uint{Key: key, Val: uint64(val)})
		return m
	}
	for _, w := range m.writers {
		w.WriteKey(key)
		w.WriteUint(uint64(val))
	}
	return m
}

func (m *Message) Uint16Ptr(key string, val *uint16) *Message {
	if val == nil {
		return m.Nil(key)
	}
	return m.Uint16(key, *val)
}

func (m *Message) Uint16s(key string, vals []uint16) *Message {
	if m == nil || m.attribs.Has(key) {
		return m
	}
	if m.IsAttribRecorder() {
		valsCpy := make([]uint64, len(vals))
		for i, val := range vals {
			valsCpy[i] = uint64(val)
		}
		m.attribs = append(m.attribs, Uints{Key: key, Vals: valsCpy})
		return m
	}
	for _, w := range m.writers {
		w.WriteSliceKey(key)
		for _, val := range vals {
			w.WriteUint(uint64(val))
		}
		w.WriteSliceEnd()
	}
	return m
}

func (m *Message) Uint32(key string, val uint32) *Message {
	if m == nil || m.attribs.Has(key) {
		return m
	}
	if m.IsAttribRecorder() {
		m.attribs = append(m.attribs, Uint{Key: key, Val: uint64(val)})
		return m
	}
	for _, w := range m.writers {
		w.WriteKey(key)
		w.WriteUint(uint64(val))
	}
	return m
}

func (m *Message) Uint32Ptr(key string, val *uint32) *Message {
	if val == nil {
		return m.Nil(key)
	}
	return m.Uint32(key, *val)
}

func (m *Message) Uint32s(key string, vals []uint32) *Message {
	if m == nil || m.attribs.Has(key) {
		return m
	}
	if m.IsAttribRecorder() {
		valsCpy := make([]uint64, len(vals))
		for i, val := range vals {
			valsCpy[i] = uint64(val)
		}
		m.attribs = append(m.attribs, Uints{Key: key, Vals: valsCpy})
		return m
	}
	for _, w := range m.writers {
		w.WriteSliceKey(key)
		for _, val := range vals {
			w.WriteUint(uint64(val))
		}
		w.WriteSliceEnd()
	}
	return m
}

func (m *Message) Uint64(key string, val uint64) *Message {
	if m == nil || m.attribs.Has(key) {
		return m
	}
	if m.IsAttribRecorder() {
		m.attribs = append(m.attribs, Uint{Key: key, Val: val})
		return m
	}
	for _, w := range m.writers {
		w.WriteKey(key)
		w.WriteUint(val)
	}
	return m
}

func (m *Message) Uint64Ptr(key string, val *uint64) *Message {
	if val == nil {
		return m.Nil(key)
	}
	return m.Uint64(key, *val)
}

func (m *Message) Uint64s(key string, vals []uint64) *Message {
	if m == nil || m.attribs.Has(key) {
		return m
	}
	if m.IsAttribRecorder() {
		valsCpy := append([]uint64(nil), vals...)
		m.attribs = append(m.attribs, Uints{Key: key, Vals: valsCpy})
		return m
	}
	for _, w := range m.writers {
		w.WriteSliceKey(key)
		for _, val := range vals {
			w.WriteUint(val)
		}
		w.WriteSliceEnd()
	}
	return m
}

func (m *Message) Float32(key string, val float32) *Message {
	if m == nil || m.attribs.Has(key) {
		return m
	}
	if m.IsAttribRecorder() {
		m.attribs = append(m.attribs, Float{Key: key, Val: float64(val)})
		return m
	}
	for _, w := range m.writers {
		w.WriteKey(key)
		w.WriteFloat(float64(val))
	}
	return m
}

func (m *Message) Float32Ptr(key string, val *float32) *Message {
	if val == nil {
		return m.Nil(key)
	}
	return m.Float32(key, *val)
}

func (m *Message) Float32s(key string, vals []float32) *Message {
	if m == nil || m.attribs.Has(key) {
		return m
	}
	if m.IsAttribRecorder() {
		valsCpy := make([]float64, len(vals))
		for i, val := range vals {
			valsCpy[i] = float64(val)
		}
		m.attribs = append(m.attribs, Floats{Key: key, Vals: valsCpy})
		return m
	}
	for _, w := range m.writers {
		w.WriteSliceKey(key)
		for _, val := range vals {
			w.WriteFloat(float64(val))
		}
		w.WriteSliceEnd()
	}
	return m
}

// Float is not called Float64 on purpose
func (m *Message) Float(key string, val float64) *Message {
	if m == nil || m.attribs.Has(key) {
		return m
	}
	if m.IsAttribRecorder() {
		m.attribs = append(m.attribs, Float{Key: key, Val: val})
		return m
	}
	for _, w := range m.writers {
		w.WriteKey(key)
		w.WriteFloat(val)
	}
	return m
}

func (m *Message) FloatPtr(key string, val *float64) *Message {
	if val == nil {
		return m.Nil(key)
	}
	return m.Float(key, *val)
}

func (m *Message) Floats(key string, vals []float64) *Message {
	if m == nil || m.attribs.Has(key) {
		return m
	}
	if m.IsAttribRecorder() {
		valsCpy := append([]float64(nil), vals...)
		m.attribs = append(m.attribs, Floats{Key: key, Vals: valsCpy})
		return m
	}
	for _, w := range m.writers {
		w.WriteSliceKey(key)
		for _, val := range vals {
			w.WriteFloat(val)
		}
		w.WriteSliceEnd()
	}
	return m
}

func (m *Message) Str(key, val string) *Message {
	if m == nil || m.attribs.Has(key) {
		return m
	}
	if m.IsAttribRecorder() {
		m.attribs = append(m.attribs, String{Key: key, Val: val})
		return m
	}
	for _, w := range m.writers {
		w.WriteKey(key)
		w.WriteString(val)
	}
	return m
}

func (m *Message) StrPtr(key string, val *string) *Message {
	if val == nil {
		return m.Nil(key)
	}
	return m.Str(key, *val)
}

func (m *Message) Strs(key string, vals []string) *Message {
	if m == nil || m.attribs.Has(key) {
		return m
	}
	if m.IsAttribRecorder() {
		valsCpy := append([]string(nil), vals...)
		m.attribs = append(m.attribs, Strings{Key: key, Vals: valsCpy})
		return m
	}
	for _, w := range m.writers {
		w.WriteSliceKey(key)
		for _, val := range vals {
			w.WriteString(val)
		}
		w.WriteSliceEnd()
	}
	return m
}

func (m *Message) Stringer(key string, val fmt.Stringer) *Message {
	if val == nil {
		return m.Nil(key)
	}
	return m.Str(key, val.String())
}

// Time logs a time.Time by calling its String method,
// or logs nil if val.IsZero().
func (m *Message) Time(key string, val time.Time) *Message {
	if val.IsZero() {
		return m.Nil(key)
	}
	return m.Str(key, val.String())
}

// TimePtr logs a time.Time by calling its String method,
// or logs nil if val is nil or val.IsZero().
func (m *Message) TimePtr(key string, val *time.Time) *Message {
	if val == nil || val.IsZero() {
		return m.Nil(key)
	}
	return m.Time(key, *val)
}

// Duration logs the passed duration as string
// using the time.Duration.String method
// representing the duration in the form "72h3m0.5s".
// Leading zero units are omitted. As a special case, durations less than one
// second format use a smaller unit (milli-, micro-, or nanoseconds) to ensure
// that the leading digit is non-zero. The zero duration formats as 0s.
func (m *Message) Duration(key string, val time.Duration) *Message {
	return m.Str(key, val.String())
}

// DurationPtr logs the passed non-nil duration as string
// using the time.Duration.String method
// representing the duration in the form "72h3m0.5s".
// Leading zero units are omitted. As a special case, durations less than one
// second format use a smaller unit (milli-, micro-, or nanoseconds) to ensure
// that the leading digit is non-zero. The zero duration formats as 0s.
// A nil duration is logged as nil.
func (m *Message) DurationPtr(key string, val *time.Duration) *Message {
	if val == nil {
		return m.Nil(key)
	}
	return m.Duration(key, *val)
}

// Millis logs the passed duration as millisecond integer.
func (m *Message) Millis(key string, val time.Duration) *Message {
	return m.Int64(key, val.Milliseconds())
}

// MillisSince logs the elapsed time since t as millisecond integer.
func (m *Message) MillisSince(key string, t time.Time) *Message {
	return m.Int64(key, time.Since(t).Microseconds())
}

// Micros logs the passed duration as microsecond integer.
func (m *Message) Micros(key string, val time.Duration) *Message {
	return m.Int64(key, val.Microseconds())
}

// MicrosSince logs the elapsed time since t as millisecond integer.
func (m *Message) MicrosSince(key string, t time.Time) *Message {
	return m.Int64(key, time.Since(t).Microseconds())
}

// UUID logs a UUID or nil in case of a "Nil UUID" containing only zero bytes.
// See IsNilUUID.
func (m *Message) UUID(key string, val [16]byte) *Message {
	if m == nil || m.attribs.Has(key) {
		return m
	}
	if m.IsAttribRecorder() {
		m.attribs = append(m.attribs, UUID{Key: key, Val: val})
		return m
	}
	if IsNilUUID(val) {
		for _, w := range m.writers {
			w.WriteKey(key)
			w.WriteNil()
		}
	} else {
		for _, w := range m.writers {
			w.WriteKey(key)
			w.WriteUUID(val)
		}
	}
	return m
}

// UUIDPtr logs a UUID or nil in case of a nil pointer or a "Nil UUID" containing only zero bytes.
// See IsNilUUID.
func (m *Message) UUIDPtr(key string, val *[16]byte) *Message {
	if val == nil {
		return m.Nil(key)
	}
	return m.UUID(key, *val)
}

// UUID logs a slice of UUIDs using nil in case of a "Nil UUID" containing only zero bytes.
// See IsNilUUID.
func (m *Message) UUIDs(key string, vals [][16]byte) *Message {
	if m == nil || m.attribs.Has(key) {
		return m
	}
	if m.IsAttribRecorder() {
		valsCpy := append([][16]byte(nil), vals...)
		m.attribs = append(m.attribs, UUIDs{Key: key, Vals: valsCpy})
		return m
	}
	for _, w := range m.writers {
		w.WriteSliceKey(key)
		for _, val := range vals {
			if IsNilUUID(val) {
				w.WriteNil()
			} else {
				w.WriteUUID(val)
			}
		}
		w.WriteSliceEnd()
	}
	return m
}

// JSON logs JSON encoded bytes
func (m *Message) JSON(key string, val []byte) *Message {
	if m == nil || m.attribs.Has(key) {
		return m
	}
	if val == nil {
		return m.Nil(key)
	}
	valCpy := bytes.NewBuffer(make([]byte, 0, len(val)))
	err := json.Compact(valCpy, val)
	if m.IsAttribRecorder() {
		if err == nil {
			m.attribs = append(m.attribs, JSON{Key: key, Val: valCpy.Bytes()})
		} else {
			m.attribs = append(m.attribs, Error{Key: key, Val: errors.New(string(val))})
		}
		return m
	}
	for _, w := range m.writers {
		w.WriteKey(key)
		if err == nil {
			w.WriteJSON(valCpy.Bytes())
		} else {
			w.WriteError(errors.New(string(val)))
		}
	}
	return m
}

// AsJSON logs the JSON marshaled val.
func (m *Message) AsJSON(key string, val any) *Message {
	if m == nil || m.attribs.Has(key) {
		return m
	}
	jsonVal, err := json.Marshal(val)
	if err != nil {
		return m.Error(key, fmt.Errorf("can't log %T AsJSON because of: %w", val, err))
	}
	if m.IsAttribRecorder() {
		m.attribs = append(m.attribs, JSON{Key: key, Val: jsonVal})
		return m
	}
	for _, w := range m.writers {
		w.WriteKey(key)
		w.WriteJSON(jsonVal)
	}
	return m
}

// Bytes logs binary data as string encoded using base64.RawURLEncoding
func (m *Message) Bytes(key string, val []byte) *Message {
	if m == nil || m.attribs.Has(key) {
		return m
	}
	if val == nil {
		return m.Nil(key)
	}
	return m.Str(key, base64.RawURLEncoding.EncodeToString(val))
}

// StrBytes logs the passed bytes as string if they are valid UTF-8,
// else the bytes are encoded using base64.RawURLEncoding.
func (m *Message) StrBytes(key string, val []byte) *Message {
	if m == nil || m.attribs.Has(key) {
		return m
	}
	if val == nil {
		return m.Nil(key)
	}
	if !utf8.Valid(val) {
		return m.Str(key, base64.RawURLEncoding.EncodeToString(val))
	}
	return m.Str(key, string(val))
}

// Request logs a http.Request including values added to the request context.
// The following request values are logged: remote, method, uri,
// and contentLength only if available and greater than zero.
// If onlyHeaders are passed, then only those headers are logged if available,
// else all headers not in the package level FilterHTTPHeaders map will be logged.
// To disable header logging, pass an impossible header name.
func (m *Message) Request(request *http.Request, onlyHeaders ...string) *Message {
	if m == nil {
		return nil
	}

	AttribsFromContext(request.Context()).Log(m)

	m.Str("remote", request.RemoteAddr)
	m.Str("method", request.Method)
	m.Str("uri", request.RequestURI)
	if request.ContentLength > 0 {
		m.Int64("Content-Length", request.ContentLength)
	}

	if len(onlyHeaders) > 0 {
		for _, header := range onlyHeaders {
			if values, ok := request.Header[header]; ok {
				if len(values) == 1 {
					m.Str(header, values[0])
				} else {
					m.Strs(header, values)
				}
			}
		}
	} else {
		for header, values := range request.Header {
			if _, filter := FilterHTTPHeaders[header]; !filter {
				if len(values) == 1 {
					m.Str(header, values[0])
				} else {
					m.Strs(header, values)
				}
			}
		}
	}

	return m
}

// Log writes the complete log message
// and returns the Message to a sync.Pool.
func (m *Message) Log() {
	if m == nil {
		return
	}

	for _, w := range m.writers {
		w.CommitMessage()
	}

	if GlobalPanicLevel.Valid() && m.level >= GlobalPanicLevel {
		panic(m.text)
	}

	// Reset and return to pools
	clear(m.writers)
	writersPool.Put(m.writers[:0])

	m.logger = nil
	m.attribs = nil
	m.writers = nil
	m.level = LevelInvalid
	m.text = ""
	messagePool.Put(m)
}

// LogAndPanic writes the complete log message
// and panics with the message text.
func (m *Message) LogAndPanic() {
	if m == nil {
		panic("nil golog.Message.LogAndPanic")
	}
	text := m.text
	m.Log() // sets m.text = ""
	panic(text)
}
