package golog

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go/token"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"time"
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

// RecordedValues returns the recorded message values and true
// if the message was created by Logger.With() for value recording.
func (m *Message) RecordedValues() (values Values, ok bool) {
	if m == nil {
		return nil, false
	}
	recorder, ok := m.formatter.(*valueRecorder)
	if !ok {
		return nil, false
	}
	return recorder.Values(), true
}

// SubLogger returns a new sub-logger with recorded per message values.
func (m *Message) SubLogger() *Logger {
	if m == nil {
		return nil
	}
	values, ok := m.RecordedValues()
	if !ok {
		panic("golog.Message was not created by golog.Logger.With()")
	}
	return m.logger.WithValues(values...)
}

// SubLoggerContext returns a new sub-logger with recorded per message values
// in addition to any values from ctx,
// and a context with those values added to it.
func (m *Message) SubLoggerContext(ctx context.Context) (subLogger *Logger, subContext context.Context) {
	if m == nil {
		return nil, ctx
	}
	recorder, ok := m.formatter.(*valueRecorder)
	if !ok {
		panic("golog.Message was not created by golog.Logger.With()")
	}
	values := MergeValues(ValuesFromContext(ctx), recorder.Values())
	subLogger = m.logger.WithValues(values...)
	subContext = values.AddToContext(ctx)
	return subLogger, subContext
}

// Ctx logs any values that were added to the context
func (m *Message) Ctx(ctx context.Context) *Message {
	if m == nil {
		return nil
	}
	ValuesFromContext(ctx).Log(m)
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

// Any logs val with the best matching typed log method
// or uses Print if none was found.
func (m *Message) Any(key string, val interface{}) *Message {
	if m == nil {
		return nil
	}
	m.formatter.WriteKey(key)
	m.writeAny(reflect.ValueOf(val), true)
	return m
}

func (m *Message) writeAny(val reflect.Value, noSlice bool) {
	// Try if val implements a loggable interface or is nil
	written := m.tryWriteInterface(val)
	if written {
		return
	}

	// Deref pointers
	valChanged := false
	for val.Kind() == reflect.Ptr && !val.IsNil() {
		val = val.Elem()
		valChanged = true
	}
	if valChanged {
		// Try if dereferenced val implements a loggable interface or is nil
		written := m.tryWriteInterface(val)
		if written {
			return
		}
	}

	switch val.Kind() {
	case reflect.Bool:
		m.formatter.WriteBool(val.Bool())

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		m.formatter.WriteInt(val.Int())

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		m.formatter.WriteUint(val.Uint())

	case reflect.Float32, reflect.Float64:
		m.formatter.WriteFloat(val.Float())

	case reflect.String:
		m.formatter.WriteString(val.String())

	case reflect.Struct, reflect.Map:
		j, err := json.Marshal(val.Interface())
		if err != nil {
			m.formatter.WriteError(fmt.Errorf("error while marshalling %s as JSON for logging: %w", val.Type(), err))
			return
		}
		m.formatter.WriteJSON(j)

	case reflect.Array, reflect.Slice:
		if noSlice {
			// TODO: why is noSlice always true?
			m.formatter.WriteString(fmt.Sprint(val))
		} else {
			for i := 0; i < val.Len(); i++ {
				m.writeAny(val.Index(i), true)
			}
		}

	default:
		m.formatter.WriteString(fmt.Sprint(val))
	}
}

func (m *Message) tryWriteInterface(val reflect.Value) (written bool) {
	switch x := val.Interface().(type) {
	case nil:
		m.formatter.WriteNil()
		return true

	case Loggable:
		x.Log(m)
		return true

	case error:
		m.formatter.WriteError(x)
		return true

	case [16]byte:
		m.formatter.WriteUUID(x)
		return true
	}

	return false
}

// StructFields calls Any(fieldName, fieldValue) for every exported struct field
func (m *Message) StructFields(strct interface{}) *Message {
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
func (m *Message) TaggedStructFields(strct interface{}, tag string) *Message {
	if m == nil || strct == nil {
		return m
	}
	m.structFields(reflect.ValueOf(strct), tag)
	return m
}

func (m *Message) structFields(v reflect.Value, tag string) {
	for v.Kind() == reflect.Ptr {
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
	return m.Str(key, val.String())
}

func (m *Message) Duration(key string, val time.Duration) *Message {
	return m.Str(key, val.String())
}

func (m *Message) DurationPtr(key string, val *time.Duration) *Message {
	if val == nil {
		return m.Nil(key)
	}
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

// JSON logs JSON encoded bytes
func (m *Message) JSON(key string, val []byte) *Message {
	if m == nil {
		return nil
	}
	m.formatter.WriteKey(key)
	buf := bytes.NewBuffer(make([]byte, 0, len(val)))
	err := json.Compact(buf, val)
	if err == nil {
		m.formatter.WriteJSON(buf.Bytes())
	} else {
		m.formatter.WriteError(errors.New(string(val)))
	}
	return m
}

// AsJSON logs the JSON marshaled val.
func (m *Message) AsJSON(key string, val interface{}) *Message {
	if m == nil {
		return nil
	}
	j, err := json.Marshal(val)
	if err != nil {
		return m.Error(key, fmt.Errorf("can't log %T AsJSON because of: %w", val, err))
	}
	m.formatter.WriteKey(key)
	m.formatter.WriteJSON(j)
	return m
}

// Request logs a http.Request including values added to the request context.
// The following request values are logged: remote, method, uri,
// and contentLength only if available and greater than zero.
// If restrictHeaders are passed, then only those headers are logged if available,
// else all headers not in the package level FilterHTTPHeaders map will be logged.
// To disable header logging, pass an impossible header name.
func (m *Message) Request(request *http.Request, restrictHeaders ...string) *Message {
	if m == nil {
		return nil
	}

	ValuesFromContext(request.Context()).Log(m)

	m.Str("remote", request.RemoteAddr)
	m.Str("method", request.Method)
	m.Str("uri", request.RequestURI)
	if request.ContentLength > 0 {
		m.Int64("Content-Length", request.ContentLength)
	}

	if len(restrictHeaders) > 0 {
		for _, header := range restrictHeaders {
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
