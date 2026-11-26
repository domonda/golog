package golog

import (
	"bytes"
	"context"
	"encoding"
	"encoding/json"
	"errors"
	"fmt"
	"go/token"
	"net/http"
	"reflect"
	"slices"
	"time"
	"unicode/utf8"

	"github.com/domonda/go-encjson"
)

// Message is a fluent builder for constructing log messages with typed attributes.
// Create messages using Logger methods like Info(), Debug(), Error(), etc.
// Add attributes with methods like Str(), Int(), Err(), etc.
// Call Log() to emit the message, or LogAndPanic() to log and panic.
// A nil Message is safe to use and will not log anything.
//
// Messages should not be reused after calling Log() or SubLogger()
// as they are returned to an internal pool.
type Message struct {
	logger  *Logger
	attribs Attribs
	// ctxAttribs are the attribs of the context that the message was logged
	// with. They are owned by the context and only borrowed, which is safe
	// because a context never returns its attribs to the mempool.
	// Together with attribs they hold the keys that were already written.
	ctxAttribs   Attribs
	writers      []Writer
	writersArray [4]Writer // Backing store for writers slice - avoids heap allocation for ≤4 writers
	level        Level
	text         string // Used for LogAndPanic
	// attribRecorder marks a Message created by Logger.With to record the
	// logged attribs for a sub-logger instead of writing them. It can't be
	// derived from an empty writers slice, because a message of a logger
	// without writer configs, or whose writers all filtered out its level,
	// has no writers either.
	attribRecorder bool
	// loggables holds the Loggables whose Log methods are logging
	// to the message, so a Loggable that logs itself, directly or
	// through other values, is logged with a cycle error instead
	// of recursing until the stack overflows.
	loggables valuePath
	// outermostWork and outermostRefs back the path that loggablePath
	// returns for the values of the message, so logging doesn't allocate.
	outermostWork valuePathWork
	outermostRefs [4]valueRef
	// writingValue is true while forEachWriter writes a value
	writingValue bool
}

// loggablePath returns the path of the Loggables that are logging
// to the message, or for the outermost value an empty path that uses
// the counters and the array of the message instead of allocating them.
// The counters are shared by all values of the message, so the limits
// of valuePath and valueCheckBudget apply to the whole log message
// instead of to each of its values, see forEachWriter for its writers.
func (m *Message) loggablePath() valuePath {
	if m.loggables.work != nil {
		return m.loggables
	}
	return valuePath{refs: m.outermostRefs[:0], work: &m.outermostWork}
}

// forEachWriter calls write for each writer of the message, each with the
// counters of loggablePath that the first writer started with, so every
// writer writes the value the same way instead of the later writers
// writing an error when the earlier ones used up the limits.
// A call nested in the write of another one, like by a Loggable that
// logs to a message that it captured, doesn't restore the counters,
// so nested calls can't multiply the work by the number of writers.
func (m *Message) forEachWriter(write func(Writer)) {
	if m.writingValue {
		for _, w := range m.writers {
			write(w)
		}
		return
	}
	m.writingValue = true
	defer func() { m.writingValue = false }()
	work := m.loggablePath().work
	start := *work
	for i, w := range m.writers {
		if i > 0 {
			*work = start
		}
		write(w)
	}
}

// newMessage returns a Message from the mempool
// that writes the logged attribs to the passed writers.
func newMessage(logger *Logger, writers []Writer, level Level, text string) *Message {
	m := messagePool.GetOrNew()
	m.logger = logger
	// Use embedded array as backing store, append handles overflow to heap if >4
	m.writers = append(m.writersArray[:0], writers...)
	m.level = level
	m.text = text
	return m
}

// newAttribRecorderMessage returns a Message from the mempool that records
// the logged attribs for a sub-logger instead of writing them,
// see Logger.With and Message.SubLogger.
// The returned Message takes over ownership of the passed attribs.
func newAttribRecorderMessage(logger *Logger, attribs Attribs) *Message {
	m := messagePool.GetOrNew()
	m.logger = logger
	m.attribs = attribs
	m.attribRecorder = true
	m.level = LevelInvalid
	return m
}

// hasKey returns true if an attrib with the passed key was already
// written or recorded, so that no key is written twice.
// It is called by every field method, so the common case
// of a message without attribs is inlined.
func (m *Message) hasKey(key string) bool {
	return (len(m.attribs) > 0 || len(m.ctxAttribs) > 0) && m.attribsHaveKey(key)
}

// attribsHaveKey returns true if the attribs or ctxAttribs
// of the message have an attrib with the passed key.
func (m *Message) attribsHaveKey(key string) bool {
	return m.attribs.Has(key) || m.ctxAttribs.Has(key)
}

func (m *Message) reset() {
	m.attribs.Free()
	// Zero the entire message (including writersArray, clearing writer refs for GC)
	var zero Message
	*m = zero
}

// IsActive returns true if the message is not nil.
func (m *Message) IsActive() bool {
	return m != nil
}

// IsAttribRecorder returns true if the Message was created by Logger.With()
// for recording attribs instead of immediately logging them.
func (m *Message) IsAttribRecorder() bool {
	return m != nil && m.attribRecorder
}

// SubLogger returns a new sub-logger with recorded per message attribs.
// Don't use the Message after calling this method,
// it will be put back into the pool.
//
// If the Message was not created by Logger.With for recording attribs,
// then the logger of the message is returned unchanged and the message is
// not put back into the pool, so that it can still be logged.
func (m *Message) SubLogger() *Logger {
	if m == nil {
		return nil
	}
	if !m.IsAttribRecorder() {
		// Message was not created by Logger.With() for recording attribs
		// which isn't how it should be used, so return the original logger
		// and don't put the message back into the pool
		return m.logger
	}
	// Attribs recorded at the message shadow the ctx attribs that the
	// parent logger inherited, so don't pass those on to the sub-logger
	return m.subLogger(m.attribs.clonesOfNonExisting(m.logger.ctxAttribs))
}

// subLogger returns a sub-logger that takes over ownership of the attribs
// recorded at the message and of the passed ctxAttribs, and puts the
// message back into the mempool.
// Only call it for a Message that is an attrib recorder.
func (m *Message) subLogger(ctxAttribs Attribs) *Logger {
	// The message has ownership of the cloned parent logger attribs
	// and those added by methods of the message.
	// Transfer attribs ownership to sub-logger by direct assignment
	// (no Clone needed since message will be reset and pooled).
	subLog := &Logger{
		config:     m.logger.config,
		prefix:     m.logger.prefix,
		attribs:    m.attribs,
		ctxAttribs: ctxAttribs,
	}
	// Nil out attribs before reset() to prevent freeing
	// the attribs that are now owned by subLog.
	m.attribs = nil
	m.reset()
	messagePool.PutBack(m)
	return subLog
}

// SubLoggerContext returns a new sub-logger with recorded per message attribs
// in addition to any attribs from the passed ctx
// and a context with those attribs added to it.
// The returned sub-logger and context have the same union of attribs
// attached to them, where the attribs recorded at the message
// take precedence over attribs from the ctx with the same key.
//
// The attribs taken from the ctx stay context attribs of the sub-logger
// instead of becoming logger attribs: they are logged after the attribs of
// the context that a message is logged with, so an attrib with the same key
// from that context takes precedence over the inherited one. A sub-logger
// created from a request scoped ctx can therefore be used with the context
// of another request without mislabeling its messages.
// See Logger.CtxAttribs.
//
// If the Message was not created by Logger.With for recording attribs, then
// the logger of the message is returned unchanged, which has none of the
// attribs, together with a context with the message attribs added to it,
// and the message is not put back into the pool so that it can still be
// logged.
func (m *Message) SubLoggerContext(ctx context.Context) (*Logger, context.Context) {
	if m == nil {
		return nil, ctx
	}
	if !m.IsAttribRecorder() {
		// Message was not created by Logger.With() for recording attribs
		// which isn't how it should be used, so return the original logger
		// and don't put the message back into the pool
		return m.logger, m.attribs.AddToContext(ctx)
	}

	// Add the ctx attribs that the parent logger inherited to the passed ctx,
	// so that the sub-logger doesn't lose them and the returned context has
	// them too. The attribs of the passed ctx are the fresher ones, so they
	// take precedence over the inherited ones.
	if inherited := m.logger.ctxAttribs; len(inherited) > 0 {
		ctxAttribs := AttribsFromContext(ctx)
		if ctxAttribs.numNonExisting(inherited) > 0 {
			ctx = context.WithValue(ctx, &attribsCtxKey, ctxAttribs.CloneAndAppendNonExistingCloned(inherited))
		}
	}

	// The context gets clones of the message attribs
	// merged with the attribs already added to the passed ctx.
	ctxWithAttribs := m.attribs.AddToContext(ctx)
	// The ctx attribs that the message attribs don't shadow become the ctx
	// attribs of the sub-logger, so that the sub-logger and the returned
	// context have the same union of attribs attached to them.
	ctxAttribs := m.attribs.clonesOfNonExisting(AttribsFromContext(ctx))

	subLog := m.subLogger(ctxAttribs) // Puts the message back into the pool

	if writerConfigsFromCtx := WriterConfigsFromContext(ctx); len(writerConfigsFromCtx) > 0 {
		writerConfigs := subLog.config.WriterConfigs()
		merged := mergeWriterConfigs(writerConfigs, writerConfigsFromCtx)
		if len(merged) != len(writerConfigs) {
			// Copy to take address or var before re-assign
			origConfig := subLog.config
			subLog.config = &DerivedConfig{
				parent:        &origConfig,
				writerConfigs: merged,
			}
		}
	}

	return subLog, ctxWithAttribs
}

// SubContext returns a new context with recorded per message attribs
// added to the passed ctx argument.
// The message keeps its attribs and can still be used.
func (m *Message) SubContext(ctx context.Context) context.Context {
	if m == nil {
		return ctx
	}
	return m.attribs.AddToContext(ctx)
}

// Ctx logs the attribs that were added to the context
// with a key that the message has not written yet,
// like a key of the logger's attribs, of the attribs the logger
// inherited from a context (see Message.SubLoggerContext),
// or of the context the message was logged with.
// Later attribs with the key of a logged context attrib are dropped.
func (m *Message) Ctx(ctx context.Context) *Message {
	if m == nil {
		return nil
	}
	ctxAttribs := AttribsFromContext(ctx)
	if m.IsAttribRecorder() {
		// The Message methods record clones of the attribs
		ctxAttribs.Log(m)
		return m
	}
	if len(m.ctxAttribs) == 0 {
		ctxAttribs.Log(m)
		// Borrow the attribs of the context like Logger.NewMessageAt does
		m.ctxAttribs = ctxAttribs
		return m
	}
	// The message borrows the attribs of one context only,
	// so it records clones of the attribs of any further context
	for _, attrib := range ctxAttribs {
		if !m.hasKey(attrib.Key()) {
			attrib.Log(m)
			m.attribs.Add(attrib.Clone())
		}
	}
	return m
}

// Loggable lets an implementation of the Loggable interface log itself
// by logging its attribs as attribs of the message.
// Use Message.Any to log them as JSON object under a key instead.
//
// A Loggable that is already logging to the message, because it logs
// itself directly or through other Loggables, is not logged again,
// and a panic of its Log method is logged with Message.Err.
// Like Message.Any writes null for them, nothing is logged for a nil
// pointer, a nil LoggableFunc, a pointer to one, or a Loggable whose
// IsNull method returns true.
func (m *Message) Loggable(loggable Loggable) *Message {
	if m == nil || loggable == nil {
		return m
	}
	v := reflect.ValueOf(loggable)
	if isNullValue(v) {
		return m
	}
	path, err := m.loggablePath().enterLoggable(v)
	if err != nil {
		return m
	}
	outer := m.loggables
	m.loggables = path
	err = safeLog(loggable, m)
	m.loggables = outer
	if err != nil {
		m.Err(err)
	}
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
	if m == nil || m.hasKey(key) {
		return m
	}
	if m.IsAttribRecorder() {
		m.attribs.Add(NewError(key, val))
		return m
	}
	if val == nil || isNilPointer(val) {
		return m.Nil(key)
	}
	for _, w := range m.writers {
		w.WriteKey(key)
		writeError(w, val)
	}
	return m
}

// Errs is a shortcut for Errors("errors", vals)
func (m *Message) Errs(vals []error) *Message {
	return m.Errors("errors", vals)
}

func (m *Message) Errors(key string, vals []error) *Message {
	if m == nil || m.hasKey(key) {
		return m
	}
	if m.IsAttribRecorder() {
		m.attribs.Add(NewErrorsCopy(key, vals))
		return m
	}
	for _, w := range m.writers {
		w.WriteSliceKey(key)
		for _, val := range vals {
			writeError(w, val)
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
// A time.Time is logged like with Message.Time,
// bytes like with Message.StrBytes,
// and a json.RawMessage like with Message.JSON.
// A byte slice type with a format of its own, like net.IP,
// is logged as one value with that format instead of as bytes:
// as error or Loggable, else with its MarshalJSON, MarshalText
// or String method in this order, like encoding/json uses them.
// The attribs that a Loggable logs are written as JSON object
// under the key, use Message.Loggable to log them as attribs
// of the message instead.
//
// A struct or map is encoded as a whole by encoding/json, so its fields
// and values follow the rules of encoding/json instead of golog's:
// a []byte in them is written as standard base64 with padding,
// also when it holds text. Use Message.StructFields to log the fields
// of a struct as attribs with golog's rules.
func (m *Message) Any(key string, val any) *Message {
	if m == nil || m.hasKey(key) {
		return m
	}
	if m.IsAttribRecorder() {
		m.attribs.Add(NewAny(key, val))
		return m
	}

	// Fast path without reflection for the most common types,
	// written like the typed methods write them
	switch x := val.(type) {
	case nil:
		return m.Nil(key)
	case string:
		return m.Str(key, x)
	case bool:
		return m.Bool(key, x)
	case int:
		return m.Int(key, x)
	case int64:
		return m.Int64(key, x)
	case float64:
		return m.Float(key, x)
	case time.Time:
		// Null for the zero time like Message.Time
		return m.Time(key, x)
	case [16]byte:
		// Null for the nil UUID like Message.UUID
		return m.UUID(key, x)
	case json.RawMessage:
		return m.JSON(key, x)
	case error:
		// Like tryWriteInterface, unless the error is also a Loggable,
		// which is written as JSON object, or is null
		if _, ok := x.(Loggable); !ok && !isNullValue(reflect.ValueOf(x)) {
			return m.Error(key, x)
		}
	}

	v := reflect.ValueOf(val)

	// Bytes are logged like Message.StrBytes instead of as slice of numbers
	if b, ok := byteSliceOf(v); ok {
		if isNullValue(v) {
			return m.Nil(key)
		}
		return m.StrBytes(key, b)
	}

	// A Loggable like Attribs is written as JSON object even if it's a slice.
	// Record its attribs once for all writers instead of once per writer.
	if loggable, ok := val.(Loggable); ok {
		if isNullValue(v) {
			return m.Nil(key)
		}
		path, err := m.loggablePath().enterLoggable(v)
		if err != nil {
			return m.Error(key, err)
		}
		j := appendLoggableJSON(nil, m.logger, loggable, path)
		for _, w := range m.writers {
			w.WriteKey(key)
			w.WriteJSON(j)
		}
		return m
	}

	// A byte slice type with a format of its own, like net.IP,
	// is one value instead of a slice of numbers
	if hasOwnByteSliceFormat(v) {
		m.forEachWriter(func(w Writer) {
			w.WriteKey(key)
			m.writeAny(w, v, false)
		})
		return m
	}

	if isSlice(v) {
		m.forEachWriter(func(w Writer) {
			w.WriteSliceKey(key)
			m.writeAny(w, v, false)
			w.WriteSliceEnd()
		})
		return m
	}

	m.forEachWriter(func(w Writer) {
		w.WriteKey(key)
		m.writeAny(w, v, false)
	})
	return m
}

// isSlice returns true if writeAny writes v, also behind pointers and
// interfaces, as the elements of a slice or array, and false if writeAny
// writes one value for it, like for a Loggable, an error or a null value
// that it finds on the way, see classifyOneValue.
func isSlice(v reflect.Value) bool {
	if !v.IsValid() {
		return false
	}
	// Check v and every pointer and interface that writeAny follows,
	// limited like writeAny, because a pointer that points to itself
	// has no end
	for step := 0; ; step++ {
		if class, _ := classifyOneValue(v); class != notOneValue {
			return false
		}
		if (v.Kind() != reflect.Pointer && v.Kind() != reflect.Interface) || step == maxAnyDerefSteps {
			return v.Kind() == reflect.Slice || (v.Kind() == reflect.Array && !isUUID(v))
		}
		v = v.Elem()
	}
}

var (
	loggableType = reflect.TypeFor[Loggable]()
	errorType    = reflect.TypeFor[error]()
)

// derefType returns the type that t points to,
// also through several pointers, or t if it is no pointer.
func derefType(t reflect.Type) reflect.Type {
	for range maxPointerLevels {
		if t.Kind() != reflect.Pointer {
			break
		}
		t = t.Elem()
	}
	return t
}

// maxAnyDerefSteps is the maximum number of pointers and interfaces
// that writeAny dereferences, so a reference cycle can't hang it.
const maxAnyDerefSteps = 8

func (m *Message) writeAny(w Writer, val reflect.Value, nestedSlice bool) {
	// Try if val implements a loggable interface or is nil
	written := m.tryWriteInterface(w, val)
	if written {
		return
	}

	// Deref pointers and unwrap interface values,
	// for example the elements of a []any.
	// Try every step for a loggable interface or nil, because an
	// intermediate pointer has the methods with pointer receiver,
	// like IsNull or Log, that the value it points to lacks.
	// The steps are limited, so a reference cycle can't hang Any.
	for range maxAnyDerefSteps {
		if (val.Kind() != reflect.Pointer && val.Kind() != reflect.Interface) || val.IsNil() {
			break
		}
		val = val.Elem()
		if m.tryWriteInterface(w, val) {
			// Also a byte slice type with a format of its own,
			// before its methods with pointer receiver are lost
			return
		}
	}

	switch val.Kind() {
	case reflect.Pointer, reflect.Interface:
		// A non-nil pointer or interface is left after
		// maxAnyDerefSteps only by a reference cycle
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
		// Like encoding/json, marshal an addressable value, like one
		// behind a pointer or an element of a slice, through its pointer,
		// so the MarshalJSON or MarshalText method with pointer receiver
		// of a type that redacts itself is used
		x := val.Interface()
		if val.CanAddr() {
			x = val.Addr().Interface()
		}
		j, err := safeJSONMarshal(x, m.loggablePath().checkBudget())
		if err != nil {
			w.WriteError(fmt.Errorf("error while marshalling %s as JSON for logging: %w", val.Type(), err))
			return
		}
		w.WriteJSON(j)

	case reflect.Array:
		if uuid, ok := asUUID(val); ok {
			writeUUID(w, uuid)
			return
		}
		if nestedSlice {
			// Don't go further into a slice of slices
			m.writeNestedSlice(w, val)
		} else {
			for i := range val.Len() {
				m.writeAny(w,
					val.Index(i),
					true, // nestedSlice
				)
			}
		}

	case reflect.Slice:
		if nestedSlice {
			// Don't go further into a slice of slices
			m.writeNestedSlice(w, val)
		} else {
			for i := range val.Len() {
				m.writeAny(w,
					val.Index(i),
					true, // nestedSlice
				)
			}
		}

	default:
		writeFormatted(w, val)
	}
}

// writeNestedSlice writes a slice or array that is an element of a slice.
// A slice of numbers, strings or bools without methods is formatted by the
// fmt package as one string. Other slices are written as JSON array like
// the Any attrib writes them, because fmt would show what the methods
// of their elements format or redact, like Error or MarshalJSON.
func (m *Message) writeNestedSlice(w Writer, val reflect.Value) {
	path := m.loggablePath()
	if formatsWithString(val.Type()) {
		// A slice or array type with a String method
		w.WriteString(safeString(val.Interface().(fmt.Stringer)))
		return
	}
	if formatsWithString(val.Type().Elem()) {
		// Like before golog v1.4.1, so a String method
		// that hides a secret keeps it hidden
		w.WriteString(sprintChecked(path.checkBudget(), val))
		return
	}
	if formatsPlainly(val.Type()) {
		// Checked with the budget of the logged value, the elements
		// can reference the same slices, see typeNeedsFormatCheck
		if err := checkFormattable(val, printVerb, path.checkBudget()); err != nil {
			w.WriteString(err.Error())
			return
		}
		// Without methods, which could panic, or interfaces, which could
		// lead back to the slice, so fmt can format it without checks
		w.WriteString(fmt.Sprint(val))
		return
	}
	// Elements written with their String method like fmt formats them
	path.inElements = true
	w.WriteJSON(appendElementsJSON(nil, val, path))
}

// formatsWithString returns true if t has a String method, but no
// MarshalJSON, MarshalText, Error or Log method, so the fmt package
// formats a value of type t with String, which encoding/json wouldn't
// use, and golog writes it with its String method in a nested slice.
func formatsWithString(t reflect.Type) bool {
	return t.NumMethod() > 0 && t.Implements(fmtStringerType) && !implementsJSONOrTextFormat(t) &&
		!t.Implements(errorType) && !t.Implements(loggableType)
}

// formatsPlainly returns true if t is a number, string or bool type
// without methods, or a slice or array of those, which the fmt package
// formats like the Any attrib writes them as JSON, or which JSON can't
// encode, like complex numbers.
func formatsPlainly(t reflect.Type) bool {
	for range maxPointerLevels {
		if t.NumMethod() > 0 || reflect.PointerTo(t).NumMethod() > 0 {
			return false
		}
		switch t.Kind() {
		case reflect.Slice, reflect.Array:
			t = t.Elem()
		case reflect.Bool,
			reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
			reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128, reflect.String:
			return true
		default:
			return false
		}
	}
	return false
}

// writeFormatted writes val formatted by the fmt package,
// or an error if fmt would format it forever or in exponential time,
// see checkFormattable.
func writeFormatted(w Writer, val reflect.Value) {
	w.WriteString(safeSprint(val))
}

// oneValue tells how a value is logged as one value,
// see classifyOneValue.
type oneValue int

const (
	// notOneValue is a value that isn't logged by its type or methods:
	// a pointer or interface is followed, a slice or array is logged
	// element by element, and other values by their kind.
	notOneValue oneValue = iota
	// oneNull is a nil value or one whose IsNull method returns true.
	oneNull
	// oneLoggable is a Loggable, logged as JSON object of its attribs.
	oneLoggable
	// oneError is an error, logged as its message.
	oneError
	// oneUUID is a [16]byte, logged as UUID, or as null if it is nil.
	oneUUID
	// oneRawJSON is a json.RawMessage, logged as JSON.
	oneRawJSON
	// oneTime is a time.Time, logged as time, or as null if it is zero.
	oneTime
	// oneOwnByteFormat is a byte slice type with a format of its own,
	// like net.IP, logged with its MarshalJSON, MarshalText or String
	// method in this order, like encoding/json uses them.
	oneOwnByteFormat
	// oneBytes is a byte slice type, logged like Message.StrBytes.
	oneBytes
	// oneMarshaler has a MarshalJSON or MarshalText method,
	// logged with it like encoding/json does, see marshalerValue.
	// A named UUID type is logged as UUID if its method formats
	// it as such, see marshalsAsUUID.
	oneMarshaler
)

// classifyOneValue returns how v is logged as one value, or notOneValue,
// and the value to log: the value that an interface holds, or the
// address of an addressable value whose methods with pointer receiver
// are used, like encoding/json uses them. It is the one place that
// decides this for Message.Any, which writes the values with
// tryWriteInterface, and for the Any attrib, which writes them as JSON
// with appendOneValueJSON, so both write the same values the same way.
func classifyOneValue(v reflect.Value) (oneValue, reflect.Value) {
	if !v.IsValid() || isNullValue(v) {
		return oneNull, v
	}
	if v.Kind() == reflect.Interface {
		// For example an element of a []any, not nil after isNullValue
		v = v.Elem()
	}
	t := v.Type()
	if k := v.Kind(); (k >= reflect.Bool && k <= reflect.Complex128 || k == reflect.String) &&
		t.NumMethod() == 0 && (!v.CanAddr() || t.PkgPath() == "") {
		// Fast path for numbers, strings and bools without methods,
		// also with pointer receiver, which are no value of the classes
		return notOneValue, v
	}
	// Like encoding/json, use the format methods with pointer receiver of
	// an addressable byte slice value, for example an element of a slice
	if v.CanAddr() && v.Kind() != reflect.Pointer && isByteSliceType(t) && implementsOwnFormat(reflect.PointerTo(t)) {
		v = v.Addr()
		t = v.Type()
	}
	switch {
	case t.Implements(loggableType):
		return oneLoggable, v
	case t.Implements(errorType):
		return oneError, v
	case t == uuidType:
		return oneUUID, v
	case t == reflect.TypeFor[json.RawMessage]():
		return oneRawJSON, v
	case t == reflect.TypeFor[time.Time]():
		return oneTime, v
	}
	if hasOwnByteSliceFormat(v) {
		if implementsJSONOrTextFormat(t) || t.Implements(fmtStringerType) {
			return oneOwnByteFormat, v
		}
		// A pointer to a pointer to a byte slice type with a format
		// of its own, followed by writeAny to the format methods
		return notOneValue, v
	}
	if _, ok := byteSliceOf(v); ok {
		return oneBytes, v
	}
	if mv, ok := marshalerValue(v); ok {
		return oneMarshaler, mv
	}
	return notOneValue, v
}

func (m *Message) tryWriteInterface(w Writer, val reflect.Value) (written bool) {
	class, val := classifyOneValue(val)
	switch class {
	case notOneValue:
		return false

	case oneNull:
		w.WriteNil()

	case oneLoggable:
		// A Loggable logs attribs with keys of its own, which can't be
		// written at the position of a value, so write them as JSON object
		path, err := m.loggablePath().enterLoggable(val)
		if err != nil {
			w.WriteError(err)
			return true
		}
		w.WriteJSON(appendLoggableJSON(nil, m.logger, val.Interface().(Loggable), path))

	case oneError:
		// A nil pointer would panic in an Error method with value receiver
		writeError(w, val.Interface().(error))

	case oneUUID:
		writeUUID(w, val.Interface().([16]byte))

	case oneRawJSON:
		// Like Message.JSON, for example for the elements of a []json.RawMessage
		writeRawJSON(w, val.Interface().(json.RawMessage))

	case oneTime:
		// Like Message.Time, using the TimeFormat and Location
		// configured for the writer instead of time.Time.MarshalJSON
		if t := val.Interface().(time.Time); t.IsZero() {
			w.WriteNil()
		} else {
			w.WriteTime(t)
		}

	case oneOwnByteFormat, oneMarshaler:
		x := val.Interface()
		if uuid, ok := asUUID(reflect.Indirect(val)); ok && marshalsAsUUID(x, uuid) {
			writeUUID(w, uuid)
			return true
		}
		switch x := x.(type) {
		case json.Marshaler:
			writeMarshalledJSON(w, x)
		case encoding.TextMarshaler:
			writeMarshalledText(w, x)
		case fmt.Stringer:
			w.WriteString(safeString(x))
		}

	case oneBytes:
		// Like Message.StrBytes, for example for the elements of a [][]byte
		if b, _ := byteSliceOf(val); b == nil {
			w.WriteNil()
		} else {
			w.WriteString(bytesAsString(b))
		}
	}
	return true
}

// writeUUID writes uuid, or null for the nil UUID like Message.UUID.
func writeUUID(w Writer, uuid [16]byte) {
	if IsNilUUID(uuid) {
		w.WriteNil()
	} else {
		w.WriteUUID(uuid)
	}
}

// marshalerValue returns val, or its address if it is addressable, if it
// has a MarshalJSON or MarshalText method, which classifyOneValue uses
// like encoding/json. Not for pointers and interfaces, which writeAny
// dereferences, and structs and maps, which writeAny marshals with
// encoding/json, which uses the methods.
func marshalerValue(val reflect.Value) (reflect.Value, bool) {
	switch val.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Struct, reflect.Map:
		return val, false
	}
	t := val.Type()
	if implementsJSONOrTextFormat(t) {
		return val, true
	}
	if val.CanAddr() && implementsJSONOrTextFormat(reflect.PointerTo(t)) {
		return val.Addr(), true
	}
	return val, false
}

// writeMarshalledJSON writes the JSON of m, or the error of m
// or of a panic of its MarshalJSON method.
func writeMarshalledJSON(w Writer, m json.Marshaler) {
	j, err := safeMarshalJSON(m)
	if err != nil {
		writeError(w, err)
	} else {
		writeRawJSON(w, j)
	}
}

// writeMarshalledText writes the text of m as string, or the error
// of m or of a panic of its MarshalText method.
func writeMarshalledText(w Writer, m encoding.TextMarshaler) {
	text, err := safeMarshalText(m)
	if err != nil {
		writeError(w, err)
	} else {
		w.WriteString(string(text))
	}
}

// StructFields logs each exported field of strct whose struct tag matches,
// in order, one of "golog", "log", "json". The first tag present on a field
// determines both the log key and which modifiers apply; other tags on the
// same field are ignored.
//
// Fields with no tag from that list are not logged. To log a field under its
// Go field name, give it an empty tag value such as `json:""` or `golog:""`.
//
// Supported modifiers (comma-separated after the name):
//   - omitempty — suppress the field when its value is empty per
//     encoding/json semantics (false, 0, nil pointer/interface, empty
//     array/slice/map/string).
//   - omitzero — suppress when the value is zero. Calls IsZero() bool if
//     the type implements it (so time.Time{} is suppressed), else falls
//     back to reflect.Value.IsZero().
//   - omitnull — suppress when the value is null. Calls IsNull() bool if
//     the type implements it (so zero golog.Timestamp is suppressed),
//     else falls back to omitzero semantics.
//   - redact (also spelled "redacted") — replace the value with
//     "***REDACTED***" in the log output. Suppression modifiers win over
//     redact: `json:",redact,omitempty"` on an empty string emits nothing,
//     not the redaction marker.
//
// Unknown modifier tokens are ignored silently, matching encoding/json's
// forward-compat posture.
//
// Breaking change vs. earlier versions: the bare form `log:",redact"`
// no longer triggers redaction — under the new unified parser it would
// name the field "redact" in the log. Migrate to `golog:",redact"`.
func (m *Message) StructFields(strct any) *Message {
	if m == nil || strct == nil {
		return m
	}
	m.structFields(reflect.ValueOf(strct), valuePath{}, nil, "golog", "log", "json")
	return m
}

// TaggedStructFields logs each exported field of strct that carries the
// given keyTag. It follows the same tag-value grammar and modifier set as
// [Message.StructFields]; see that method's documentation for details.
//
// Fields without the keyTag are not logged. Fields where the keyTag value is
// exactly "-" are skipped. A tag value with an empty name segment (e.g.
// `json:""` or `json:",omitempty"`) falls back to the Go field name, matching
// encoding/json.Marshal.
//
// Passing the empty string as keyTag is a wildcard: every exported field is
// logged under its Go field name, regardless of whether it carries any
// struct tag. This restores the pre-tag-driven "log every exported field"
// behavior for callers that want it — `TaggedStructFields(s, "")`.
func (m *Message) TaggedStructFields(strct any, keyTag string) *Message {
	if m == nil || strct == nil {
		return m
	}
	m.structFields(reflect.ValueOf(strct), valuePath{}, nil, keyTag)
	return m
}

// structFields logs the fields of the struct v, also behind pointers.
// path holds the pointers to the structs that are logged around v,
// so a struct that embeds a pointer to itself, directly or through
// other embedded structs, is logged once instead of recursing until
// the stack overflows. Like encoding/json, which logs the fields of an
// embedded struct type only once, the repeated struct is left out
// instead of being logged as error. visited holds the embedded pointers
// that were followed, so a struct that several embedded pointers point
// to is logged once instead of once for every path to it, which would
// take exponential time. It returns visited, which it allocates for
// the first embedded pointer.
func (m *Message) structFields(v reflect.Value, path valuePath, visited map[valueRef]struct{}, keyTags ...string) map[valueRef]struct{} {
	path, err := path.enter(v)
	if err != nil {
		// Already logging the struct, or embedded too deep
		return visited
	}
	// Limited, a pointer that points to itself has no end
	for range maxPointerLevels {
		if v.Kind() != reflect.Pointer {
			break
		}
		if v.IsNil() {
			return visited
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		// A pointer cycle or no struct, which has no fields
		return visited
	}
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		if field.Anonymous && derefType(field.Type).Kind() == reflect.Struct {
			fv := v.Field(i)
			if fv.Kind() == reflect.Pointer && !fv.IsNil() {
				ref := valueRef{ptr: fv.Pointer(), typ: fv.Type()}
				if _, ok := visited[ref]; ok {
					continue
				}
				if visited == nil {
					visited = make(map[valueRef]struct{})
				}
				visited[ref] = struct{}{}
			}
			visited = m.structFields(fv, path, visited, keyTags...)
			continue
		}
		// Like encoding/json, an embedded field of another type,
		// like an embedded interface or string type,
		// is a regular field under its type name if that is exported
		if !token.IsExported(field.Name) {
			continue
		}

		// Find the first keyTag that "matches" this field. Lookup (not
		// Get) so that an explicit empty tag value like `json:""` counts
		// as present.
		//
		// An empty string in keyTags is a wildcard: it matches every
		// exported field with an empty raw tag value (which the caller
		// later substitutes with the Go field name). This gives callers
		// a way to dump a struct wholesale under Go field names, e.g.
		// `TaggedStructFields(s, "")` for log-everything, or
		// `structFields(v, "json", "")` for "prefer json tag, fall back
		// to Go field name".
		var rawTag string
		var found bool
		for _, kt := range keyTags {
			if kt == "" {
				found = true
				rawTag = ""
				break
			}
			if tv, ok := field.Tag.Lookup(kt); ok {
				rawTag = tv
				found = true
				break
			}
		}
		if !found {
			continue
		}

		d := parseStructFieldDirectives(rawTag)
		if d.skip {
			continue
		}
		if d.key == "" {
			d.key = field.Name
		}

		fv := v.Field(i)
		if shouldOmitStructField(fv, d) {
			continue
		}

		if d.redact {
			m.Str(d.key, "***REDACTED***")
			continue
		}

		m.structField(d.key, fv)
	}
	return visited
}

// structField logs the struct field fv under key with Message.Any.
//
// Like encoding/json, an addressable field, of a struct passed as pointer,
// is logged through its address, so the MarshalJSON and MarshalText
// methods with pointer receiver of its type, and of the fields and elements
// in it, are used, which a type uses to redact a secret for example.
// An attrib recorder records the address of a copy, which doesn't
// reference the struct after logging it. Only if the pointer type has no
// Log, Error or IsNull method that the field type lacks, because those
// are never called for a value: then such a field is marshalled through
// its address by encoding/json, which never calls them.
func (m *Message) structField(key string, fv reflect.Value) {
	t := fv.Type()
	if !fv.CanAddr() || fv.Kind() == reflect.Pointer || fv.Kind() == reflect.Interface ||
		(fv.Kind() != reflect.Struct && fv.Kind() != reflect.Array &&
			(implementsJSONOrTextFormat(t) || !implementsJSONOrTextFormat(reflect.PointerTo(t)))) {
		m.Any(key, fv.Interface())
		return
	}
	if !addsLoggableMethods(t) {
		ptr := fv.Addr()
		if m.IsAttribRecorder() {
			ptr = reflect.New(t)
			ptr.Elem().Set(fv)
		}
		m.Any(key, ptr.Interface())
		return
	}
	if t.Implements(loggableType) || t.Implements(errorType) || isNullValue(fv) || t == reflect.TypeFor[time.Time]() {
		// Logged as such like Message.Any does
		m.Any(key, fv.Interface())
		return
	}
	j, err := safeJSONMarshal(fv.Addr().Interface(), m.loggablePath().checkBudget())
	if err != nil {
		m.Error(key, fmt.Errorf("error while marshalling %s as JSON for logging: %w", t, err))
		return
	}
	m.JSON(key, j)
}

// addsLoggableMethods returns true if the pointer type of t has
// a Log, Error or IsNull method that t lacks.
func addsLoggableMethods(t reflect.Type) bool {
	pt := reflect.PointerTo(t)
	for _, i := range []reflect.Type{loggableType, errorType, isNullIfaceType} {
		if !t.Implements(i) && pt.Implements(i) {
			return true
		}
	}
	return false
}

// implementsJSONOrTextFormat returns true if t implements
// json.Marshaler or encoding.TextMarshaler.
func implementsJSONOrTextFormat(t reflect.Type) bool {
	if t.NumMethod() == 0 {
		return false // Fast path for most types
	}
	return t.Implements(jsonMarshalerType) || t.Implements(textMarshalerType)
}

var (
	jsonMarshalerType = reflect.TypeFor[json.Marshaler]()
	textMarshalerType = reflect.TypeFor[encoding.TextMarshaler]()
)

// Print logs vals as string with the "%v" format of the fmt package.
// If only one value is passed for vals, then it will be logged as single string,
// else a slice of strings will be logged for vals.
func (m *Message) Print(key string, vals ...any) *Message {
	if m == nil || m.hasKey(key) {
		return m
	}
	// Formatted safely once for all writers with the budget
	// of the message, see sprintChecked
	budget := m.loggablePath().checkBudget()
	if len(vals) == 1 {
		str := sprintChecked(budget, vals...)
		if m.IsAttribRecorder() {
			m.attribs.Add(NewString(key, str))
			return m
		}
		for _, w := range m.writers {
			w.WriteKey(key)
			w.WriteString(str)
		}
		return m
	}
	strs := make([]string, len(vals))
	for i, val := range vals {
		strs[i] = sprintChecked(budget, val)
	}
	if m.IsAttribRecorder() {
		m.attribs.Add(NewStrings(key, strs))
		return m
	}
	for _, w := range m.writers {
		w.WriteSliceKey(key)
		for _, str := range strs {
			w.WriteString(str)
		}
		w.WriteSliceEnd()
	}
	return m
}

// Nil logs a nil value for key.
func (m *Message) Nil(key string) *Message {
	if m == nil || m.hasKey(key) {
		return m
	}
	if m.IsAttribRecorder() {
		m.attribs.Add(NewNil(key))
		return m
	}
	for _, w := range m.writers {
		w.WriteKey(key)
		w.WriteNil()
	}
	return m
}

// Bool logs val as a boolean.
// val can be a bool or any type with bool as underlying type.
func (m *Message) Bool[T ~bool](key string, val T) *Message {
	if m == nil || m.hasKey(key) {
		return m
	}
	if m.IsAttribRecorder() {
		m.attribs.Add(NewBool(key, bool(val)))
		return m
	}
	for _, w := range m.writers {
		w.WriteKey(key)
		w.WriteBool(bool(val))
	}
	return m
}

// BoolPtr logs *val as a boolean, or nil if val is nil.
// val can point to a bool or any type with bool as underlying type.
// For a literal nil call Nil(key) instead: an untyped nil can't infer the type parameter.
func (m *Message) BoolPtr[T ~bool](key string, val *T) *Message {
	if val == nil {
		return m.Nil(key)
	}
	return m.Bool(key, *val)
}

// Bools logs vals as a slice of booleans.
// vals can be a slice of any type with bool as underlying type.
func (m *Message) Bools[T ~bool](key string, vals []T) *Message {
	if m == nil || m.hasKey(key) {
		return m
	}
	if m.IsAttribRecorder() {
		m.attribs.Add(NewBoolsCopy(key, vals))
		return m
	}
	for _, w := range m.writers {
		w.WriteSliceKey(key)
		for _, val := range vals {
			w.WriteBool(bool(val))
		}
		w.WriteSliceEnd()
	}
	return m
}

// Int logs val as an integer.
// val can be of any signed integer type, including named integer types.
// Int writes the number even if the type has its own format, like a
// String method: Int("timeout", time.Second) logs 1000000000.
// Use Duration for a time.Duration to log "1s", Stringer for the result
// of a String method, or Loggable or Any for a Loggable.
func (m *Message) Int[T ~int | ~int8 | ~int16 | ~int32 | ~int64](key string, val T) *Message {
	if m == nil || m.hasKey(key) {
		return m
	}
	if m.IsAttribRecorder() {
		m.attribs.Add(NewInt(key, int64(val)))
		return m
	}
	for _, w := range m.writers {
		w.WriteKey(key)
		w.WriteInt(int64(val))
	}
	return m
}

// IntPtr logs *val as an integer, or nil if val is nil.
// val can point to any signed integer type.
// For a literal nil call Nil(key) instead: an untyped nil can't infer the type parameter.
func (m *Message) IntPtr[T ~int | ~int8 | ~int16 | ~int32 | ~int64](key string, val *T) *Message {
	if val == nil {
		return m.Nil(key)
	}
	return m.Int(key, *val)
}

// Ints logs vals as a slice of integers.
// vals can be a slice of any signed integer type.
func (m *Message) Ints[T ~int | ~int8 | ~int16 | ~int32 | ~int64](key string, vals []T) *Message {
	if m == nil || m.hasKey(key) {
		return m
	}
	if m.IsAttribRecorder() {
		m.attribs.Add(NewIntsCopy(key, vals))
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

// Int8 logs val as an integer, like Int.
func (m *Message) Int8(key string, val int8) *Message {
	return m.Int(key, val)
}

// Int8Ptr logs *val as an integer, or nil if val is nil, like IntPtr.
func (m *Message) Int8Ptr(key string, val *int8) *Message {
	return m.IntPtr(key, val)
}

// Int8s logs vals as a slice of integers, like Ints.
func (m *Message) Int8s(key string, vals []int8) *Message {
	return m.Ints(key, vals)
}

// Int16 logs val as an integer, like Int.
func (m *Message) Int16(key string, val int16) *Message {
	return m.Int(key, val)
}

// Int16Ptr logs *val as an integer, or nil if val is nil, like IntPtr.
func (m *Message) Int16Ptr(key string, val *int16) *Message {
	return m.IntPtr(key, val)
}

// Int16s logs vals as a slice of integers, like Ints.
func (m *Message) Int16s(key string, vals []int16) *Message {
	return m.Ints(key, vals)
}

// Int32 logs val as an integer, like Int.
func (m *Message) Int32(key string, val int32) *Message {
	return m.Int(key, val)
}

// Int32Ptr logs *val as an integer, or nil if val is nil, like IntPtr.
func (m *Message) Int32Ptr(key string, val *int32) *Message {
	return m.IntPtr(key, val)
}

// Int32s logs vals as a slice of integers, like Ints.
func (m *Message) Int32s(key string, vals []int32) *Message {
	return m.Ints(key, vals)
}

// Int64 logs val as an integer, like Int.
func (m *Message) Int64(key string, val int64) *Message {
	return m.Int(key, val)
}

// Int64Ptr logs *val as an integer, or nil if val is nil, like IntPtr.
func (m *Message) Int64Ptr(key string, val *int64) *Message {
	return m.IntPtr(key, val)
}

// Int64s logs vals as a slice of integers, like Ints.
func (m *Message) Int64s(key string, vals []int64) *Message {
	return m.Ints(key, vals)
}

// Uint logs val as an unsigned integer.
// val can be of any unsigned integer type, including uintptr and named
// unsigned integer types. Uint writes the number even if the type has
// its own format, like a String method: Uint("mode", os.ModePerm) logs 511.
// Use Stringer to log the result of a String method, like "-rwxrwxrwx",
// or Loggable or Any for a Loggable.
func (m *Message) Uint[T ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr](key string, val T) *Message {
	if m == nil || m.hasKey(key) {
		return m
	}
	if m.IsAttribRecorder() {
		m.attribs.Add(NewUint(key, uint64(val)))
		return m
	}
	for _, w := range m.writers {
		w.WriteKey(key)
		w.WriteUint(uint64(val))
	}
	return m
}

// UintPtr logs *val as an unsigned integer, or nil if val is nil.
// val can point to any unsigned integer type.
// For a literal nil call Nil(key) instead: an untyped nil can't infer the type parameter.
func (m *Message) UintPtr[T ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr](key string, val *T) *Message {
	if val == nil {
		return m.Nil(key)
	}
	return m.Uint(key, *val)
}

// Uints logs vals as a slice of unsigned integers.
// vals can be a slice of any unsigned integer type.
func (m *Message) Uints[T ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr](key string, vals []T) *Message {
	if m == nil || m.hasKey(key) {
		return m
	}
	if m.IsAttribRecorder() {
		m.attribs.Add(NewUintsCopy(key, vals))
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

// Uint8 logs val as an unsigned integer, like Uint.
func (m *Message) Uint8(key string, val uint8) *Message {
	return m.Uint(key, val)
}

// Uint8Ptr logs *val as an unsigned integer, or nil if val is nil, like UintPtr.
func (m *Message) Uint8Ptr(key string, val *uint8) *Message {
	return m.UintPtr(key, val)
}

// Uint8s logs the values as a slice of numbers.
//
// A []uint8 is the same Go type as []byte, but Uint8s never interprets the
// values as text or binary data. All other methods logging bytes do:
// use Message.StrBytes for bytes that hold text (the meaning Message.Any
// uses for bytes), Message.Bytes for binary data and Message.JSON for JSON.
func (m *Message) Uint8s(key string, vals []uint8) *Message {
	return m.Uints(key, vals)
}

// Uint16 logs val as an unsigned integer, like Uint.
func (m *Message) Uint16(key string, val uint16) *Message {
	return m.Uint(key, val)
}

// Uint16Ptr logs *val as an unsigned integer, or nil if val is nil, like UintPtr.
func (m *Message) Uint16Ptr(key string, val *uint16) *Message {
	return m.UintPtr(key, val)
}

// Uint16s logs vals as a slice of unsigned integers, like Uints.
func (m *Message) Uint16s(key string, vals []uint16) *Message {
	return m.Uints(key, vals)
}

// Uint32 logs val as an unsigned integer, like Uint.
func (m *Message) Uint32(key string, val uint32) *Message {
	return m.Uint(key, val)
}

// Uint32Ptr logs *val as an unsigned integer, or nil if val is nil, like UintPtr.
func (m *Message) Uint32Ptr(key string, val *uint32) *Message {
	return m.UintPtr(key, val)
}

// Uint32s logs vals as a slice of unsigned integers, like Uints.
func (m *Message) Uint32s(key string, vals []uint32) *Message {
	return m.Uints(key, vals)
}

// Uint64 logs val as an unsigned integer, like Uint.
func (m *Message) Uint64(key string, val uint64) *Message {
	return m.Uint(key, val)
}

// Uint64Ptr logs *val as an unsigned integer, or nil if val is nil, like UintPtr.
func (m *Message) Uint64Ptr(key string, val *uint64) *Message {
	return m.UintPtr(key, val)
}

// Uint64s logs vals as a slice of unsigned integers, like Uints.
func (m *Message) Uint64s(key string, vals []uint64) *Message {
	return m.Uints(key, vals)
}

// Float32 logs val as a floating point number, like Float.
func (m *Message) Float32(key string, val float32) *Message {
	return m.Float(key, val)
}

// Float32Ptr logs *val as a floating point number, or nil if val is nil, like FloatPtr.
func (m *Message) Float32Ptr(key string, val *float32) *Message {
	return m.FloatPtr(key, val)
}

// Float32s logs vals as a slice of floating point numbers, like Floats.
func (m *Message) Float32s(key string, vals []float32) *Message {
	return m.Floats(key, vals)
}

// Float logs val as a floating point number.
// val can be of any floating point type, it is written as float64.
// Float is not called Float64 on purpose.
func (m *Message) Float[T ~float32 | ~float64](key string, val T) *Message {
	if m == nil || m.hasKey(key) {
		return m
	}
	if m.IsAttribRecorder() {
		m.attribs.Add(NewFloat(key, float64(val)))
		return m
	}
	for _, w := range m.writers {
		w.WriteKey(key)
		w.WriteFloat(float64(val))
	}
	return m
}

// FloatPtr logs *val as a floating point number, or nil if val is nil.
// val can point to any floating point type.
// For a literal nil call Nil(key) instead: an untyped nil can't infer the type parameter.
func (m *Message) FloatPtr[T ~float32 | ~float64](key string, val *T) *Message {
	if val == nil {
		return m.Nil(key)
	}
	return m.Float(key, *val)
}

// Floats logs vals as a slice of floating point numbers.
// vals can be a slice of any floating point type.
func (m *Message) Floats[T ~float32 | ~float64](key string, vals []T) *Message {
	if m == nil || m.hasKey(key) {
		return m
	}
	if m.IsAttribRecorder() {
		m.attribs.Add(NewFloatsCopy(key, vals))
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

// Str logs val as a string.
// val can be a string or any type with string as underlying type,
// so named string types don't need a conversion at the call site.
// Str writes the string even if the type has its own format: a password
// type that redacts itself in a Log or String method is logged in plain
// text. Use Loggable or Any for a Loggable, Stringer for the result of
// a String method, or Any for a MarshalJSON or MarshalText method.
func (m *Message) Str[S ~string](key string, val S) *Message {
	if m == nil || m.hasKey(key) {
		return m
	}
	if m.IsAttribRecorder() {
		m.attribs.Add(NewString(key, string(val)))
		return m
	}
	for _, w := range m.writers {
		w.WriteKey(key)
		w.WriteString(string(val))
	}
	return m
}

// StrMax logs the string val with a maximum number of maxNumRunes runes.
// If maxNumRunes is <= 0 then the string is logged as is.
// val can be a string or any type with string as underlying type.
func (m *Message) StrMax[S ~string](key string, val S, maxNumRunes int) *Message {
	if maxNumRunes <= 0 {
		return m.Str(key, val)
	}
	// Skip the rune scan if the message is nil or the key is set,
	// like StrBytesMax
	if m == nil || m.hasKey(key) {
		return m
	}
	numRunes := 0
	for byteIndex := range string(val) {
		numRunes++
		if numRunes > maxNumRunes {
			return m.Str(key, val[:byteIndex])
		}
	}
	return m.Str(key, val)
}

// StrPtr logs *val as a string, or nil if val is nil.
// val can point to a string or any type with string as underlying type.
// For a literal nil call Nil(key) instead: an untyped nil can't infer the type parameter.
func (m *Message) StrPtr[S ~string](key string, val *S) *Message {
	if val == nil {
		return m.Nil(key)
	}
	return m.Str(key, *val)
}

// Strs logs vals as a slice of strings.
// vals can be a slice of any type with string as underlying type.
func (m *Message) Strs[S ~string](key string, vals []S) *Message {
	if m == nil || m.hasKey(key) {
		return m
	}
	if m.IsAttribRecorder() {
		m.attribs.Add(NewStringsCopy(key, vals))
		return m
	}
	for _, w := range m.writers {
		w.WriteSliceKey(key)
		for _, val := range vals {
			w.WriteString(string(val))
		}
		w.WriteSliceEnd()
	}
	return m
}

// Stringer logs the result of the String method of val,
// null for a nil val or a nil pointer whose String method panics,
// or an error if the String method of a value panics.
func (m *Message) Stringer(key string, val fmt.Stringer) *Message {
	if m == nil || m.hasKey(key) {
		return m
	}
	if val == nil {
		return m.Nil(key)
	}
	if isNilPointer(val) {
		// A String method with pointer receiver can handle nil,
		// but like the fmt package, which formats it as <nil>,
		// a nil pointer whose String method panics is null
		if str, ok := nilPointerString(val); ok {
			return m.Str(key, str)
		}
		return m.Nil(key)
	}
	return m.Str(key, safeString(val))
}

// nilPointerString returns the result of the String method of s,
// which is a nil pointer, or false if it panics.
func nilPointerString(s fmt.Stringer) (str string, ok bool) {
	defer func() {
		if recover() != nil {
			str, ok = "", false
		}
	}()
	return s.String(), true
}

// Time logs a time.Time formatted using the configured TimeFormat,
// or logs nil if val.IsZero().
func (m *Message) Time(key string, val time.Time) *Message {
	if m == nil || m.hasKey(key) {
		return m
	}
	if val.IsZero() {
		return m.Nil(key)
	}
	if m.IsAttribRecorder() {
		m.attribs.Add(NewTime(key, val))
		return m
	}
	for _, w := range m.writers {
		w.WriteKey(key)
		w.WriteTime(val)
	}
	return m
}

// TimePtr logs a time.Time formatted using the configured TimeFormat,
// or logs nil if val is nil or val.IsZero().
func (m *Message) TimePtr(key string, val *time.Time) *Message {
	if val == nil || val.IsZero() {
		return m.Nil(key)
	}
	return m.Time(key, *val)
}

// Times logs a slice of time.Time formatted using the configured TimeFormat.
func (m *Message) Times(key string, vals []time.Time) *Message {
	if m == nil || m.hasKey(key) {
		return m
	}
	if len(vals) == 0 {
		return m.Nil(key)
	}
	if m.IsAttribRecorder() {
		m.attribs.Add(NewTimesCopy(key, vals))
		return m
	}
	for _, w := range m.writers {
		w.WriteSliceKey(key)
		for _, val := range vals {
			w.WriteTime(val)
		}
		w.WriteSliceEnd()
	}
	return m
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
	return m.Int64(key, time.Since(t).Milliseconds())
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
	if m == nil || m.hasKey(key) {
		return m
	}
	if m.IsAttribRecorder() {
		m.attribs.Add(NewUUID(key, val))
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
// val can point to any type with [16]byte as underlying type.
// See IsNilUUID.
// For a literal nil call Nil(key) instead: an untyped nil can't infer the type parameter.
func (m *Message) UUIDPtr[U ~[16]byte](key string, val *U) *Message {
	if val == nil {
		return m.Nil(key)
	}
	return m.UUID(key, *val)
}

// UUIDs logs a slice of UUIDs using nil in case of a "Nil UUID" containing only zero bytes.
// vals can be a slice of any type with [16]byte as underlying type.
// See IsNilUUID.
func (m *Message) UUIDs[U ~[16]byte](key string, vals []U) *Message {
	if m == nil || m.hasKey(key) {
		return m
	}
	if m.IsAttribRecorder() {
		m.attribs.Add(NewUUIDsCopy(key, vals))
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

// JSON logs JSON encoded bytes as JSON value, compacted,
// nil as null, and invalid JSON as error.
// A json.RawMessage logged with Message.Any is logged the same way.
//
// Use Message.StrBytes for bytes that hold text, Message.Bytes for
// binary data, and Message.Uint8s to log the bytes as numbers.
func (m *Message) JSON(key string, val []byte) *Message {
	if m == nil || m.hasKey(key) {
		return m
	}
	if val == nil {
		return m.Nil(key)
	}
	valCpy := bytes.NewBuffer(make([]byte, 0, len(val)))
	err := json.Compact(valCpy, val)
	if m.IsAttribRecorder() {
		if err == nil {
			m.attribs.Add(NewJSON(key, valCpy.Bytes()))
		} else {
			m.attribs.Add(NewError(key, errors.New(string(val))))
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

// writeRawJSON writes val compacted like Message.JSON,
// null for nil, or val as error if it is no valid JSON.
func writeRawJSON(w Writer, val json.RawMessage) {
	if val == nil {
		w.WriteNil()
		return
	}
	var compacted bytes.Buffer
	if err := json.Compact(&compacted, val); err != nil {
		w.WriteError(errors.New(string(val)))
		return
	}
	w.WriteJSON(compacted.Bytes())
}

// AsJSON logs the JSON marshaled val.
func (m *Message) AsJSON(key string, val any) *Message {
	if m == nil || m.hasKey(key) {
		return m
	}
	jsonVal, err := safeJSONMarshal(val, m.loggablePath().checkBudget())
	if err != nil {
		return m.Error(key, fmt.Errorf("can't log %T AsJSON because of: %w", val, err))
	}
	if m.IsAttribRecorder() {
		m.attribs.Add(NewJSON(key, jsonVal))
		return m
	}
	for _, w := range m.writers {
		w.WriteKey(key)
		w.WriteJSON(jsonVal)
	}
	return m
}

// Bytes logs binary data as string encoded with BinaryEncoder,
// even if the bytes are valid UTF-8 text, and nil as null.
//
// Use Message.StrBytes for bytes that hold text, Message.JSON for JSON,
// and Message.Uint8s to log the bytes as numbers.
func (m *Message) Bytes(key string, val []byte) *Message {
	if m == nil || m.hasKey(key) {
		return m
	}
	if val == nil {
		return m.Nil(key)
	}
	return m.Str(key, BinaryEncoder(val))
}

// StrBytes logs the passed bytes as string if they are valid UTF-8,
// else the bytes are encoded with BinaryEncoder,
// and nil as null.
//
// This is the meaning of bytes everywhere in golog, for example for bytes
// logged with Message.Any or as struct field with Message.StructFields.
// Use Message.Bytes to always log binary data encoded with BinaryEncoder,
// Message.StrBytesMax to truncate the string, Message.JSON for JSON,
// and Message.Uint8s to log the bytes as numbers.
func (m *Message) StrBytes(key string, val []byte) *Message {
	if m == nil || m.hasKey(key) {
		return m
	}
	if val == nil {
		return m.Nil(key)
	}
	return m.Str(key, bytesAsString(val))
}

// appendLoggableJSON appends the attribs that the loggable logs as JSON
// object to buf, or JSON null if it logs no attribs. The attribs are
// recorded with an attrib recorder message of logger, which can be nil,
// instead of being written to any writer.
func appendLoggableJSON(buf []byte, logger *Logger, loggable Loggable, path valuePath) []byte {
	recorder := newAttribRecorderMessage(logger, nil)
	defer func() {
		recorder.reset() // Frees the recorded attribs
		messagePool.PutBack(recorder)
	}()
	recorder.loggables = path
	if err := safeLog(loggable, recorder); err != nil {
		return encjson.AppendString(buf, err.Error())
	}
	// The values of Any attribs with the path,
	// so a value that leads back to loggable is written as error
	return recorder.attribs.appendJSON(buf, path)
}

// isNullValue returns true for a nil pointer or function,
// a pointer to a nil function, or a value whose IsNull method returns true,
// which tryWriteInterface writes as null.
func isNullValue(v reflect.Value) bool {
	if v.Kind() == reflect.Interface {
		// For example an element of a []any
		if v.IsNil() {
			return true
		}
		v = v.Elem()
	}
	if (v.Kind() == reflect.Pointer || v.Kind() == reflect.Func) && v.IsNil() {
		// Never call an IsNull method with value receiver on a nil pointer,
		// and never call a nil LoggableFunc
		return true
	}
	if v.Kind() == reflect.Pointer && v.Elem().Kind() == reflect.Func && v.Elem().IsNil() {
		// A *LoggableFunc is a Loggable that would call the nil function
		return true
	}
	if !v.Type().Implements(isNullIfaceType) {
		// Don't copy the value with Interface for nothing
		return false
	}
	nullable, ok := v.Interface().(interface{ IsNull() bool })
	return ok && safeIsNull(nullable)
}

// byteSliceOf returns the bytes of v and true if v is a slice of bytes,
// including named slice and element types, or a pointer to one,
// also when wrapped in an interface value.
// A nil pointer returns nil bytes. A json.RawMessage is no byte slice
// in this sense, because it holds JSON instead of text or binary data,
// and neither is a type with a format of its own like net.IP,
// see hasOwnByteSliceFormat.
func byteSliceOf(v reflect.Value) (b []byte, ok bool) {
	if v.Kind() == reflect.Interface {
		// For example an element of a []any
		if v.IsNil() {
			return nil, false
		}
		v = v.Elem()
	}
	if !isByteSliceType(v.Type()) || hasOwnFormat(v.Type()) {
		return nil, false
	}
	for v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil, true
		}
		v = v.Elem()
	}
	return v.Bytes(), true
}

// hasOwnByteSliceFormat returns true if v, also when wrapped in an
// interface value, is a byte slice type or a pointer to one that formats
// itself, like net.IP, see hasOwnFormat.
// Such a value is no text or binary data that byteSliceOf returns.
func hasOwnByteSliceFormat(v reflect.Value) bool {
	if v.Kind() == reflect.Interface {
		if v.IsNil() {
			return false
		}
		v = v.Elem()
	}
	return v.IsValid() && isByteSliceType(v.Type()) && hasOwnFormat(v.Type())
}

// isByteSliceType returns true if t is a slice of bytes,
// including named slice and element types, or a pointer to one,
// except json.RawMessage.
func isByteSliceType(t reflect.Type) bool {
	for range maxPointerLevels {
		if t.Kind() != reflect.Pointer {
			return t.Kind() == reflect.Slice && t.Elem().Kind() == reflect.Uint8 && t != reflect.TypeFor[json.RawMessage]()
		}
		t = t.Elem()
	}
	return false
}

// maxPointerLevels limits how many pointers are followed to the type
// they point to, because pointer types can point to each other in
// a cycle like type A *B and type B *A, which has no end.
const maxPointerLevels = 16

// ownFormatInterfaces are the interfaces of a type that formats itself
// for logging or encoding/json, which is used instead of its bytes.
var ownFormatInterfaces = []reflect.Type{
	reflect.TypeFor[Loggable](),
	reflect.TypeFor[error](),
	reflect.TypeFor[json.Marshaler](),
	reflect.TypeFor[encoding.TextMarshaler](),
	reflect.TypeFor[fmt.Stringer](),
}

// hasOwnFormat returns true if t or a type that t points to,
// also through several pointers, implements one of ownFormatInterfaces.
func hasOwnFormat(t reflect.Type) bool {
	for range maxPointerLevels {
		if implementsOwnFormat(t) {
			return true
		}
		if t.Kind() != reflect.Pointer {
			return false
		}
		t = t.Elem()
	}
	return false
}

// implementsOwnFormat returns true if t implements
// one of ownFormatInterfaces, without following pointers.
func implementsOwnFormat(t reflect.Type) bool {
	if t.NumMethod() == 0 {
		return false // Fast path for []byte without methods
	}
	return slices.ContainsFunc(ownFormatInterfaces, t.Implements)
}

// bytesAsString returns the bytes as string if they are valid UTF-8,
// else encoded with BinaryEncoder, see Message.StrBytes.
func bytesAsString(val []byte) string {
	if !utf8.Valid(val) {
		return BinaryEncoder(val)
	}
	return string(val)
}

// StrBytesMax logs the passed bytes as string if they are valid UTF-8,
// else the bytes are encoded with BinaryEncoder.
// The logged string is truncated at maxNumRunes runes.
// Only the first maxNumRunes runes are checked for valid UTF-8,
// so a text cut within a rune after them is still logged as text.
// If maxNumRunes is <= 0 then the bytes are logged as is.
// See Message.StrBytes.
func (m *Message) StrBytesMax(key string, val []byte, maxNumRunes int) *Message {
	if maxNumRunes <= 0 {
		return m.StrBytes(key, val)
	}
	if m == nil || m.hasKey(key) {
		return m
	}
	if val == nil {
		return m.Nil(key)
	}
	// Decode only up to maxNumRunes runes instead of checking
	// and converting all bytes, which can be a large request body
	numRunes := 0
	for byteIndex := 0; byteIndex < len(val); {
		r, size := utf8.DecodeRune(val[byteIndex:])
		if r == utf8.RuneError && size <= 1 {
			// Invalid UTF-8, a valid U+FFFD has a size of 3.
			// Encode only as many bytes as needed for maxNumRunes
			// characters, assuming that BinaryEncoder encodes
			// every byte with at least one character.
			return m.StrMax(key, BinaryEncoder(val[:min(len(val), maxNumRunes)]), maxNumRunes)
		}
		numRunes++
		if numRunes > maxNumRunes {
			return m.Str(key, string(val[:byteIndex]))
		}
		byteIndex += size
	}
	return m.Str(key, string(val))
}

// Request logs a http.Request including values added to the request context.
//
// The following request values are logged:
//   - "remote" (request.RemoteAddr)
//   - "method" (request.Method)
//   - "uri" (request.RequestURI)
//   - "contentLength" (request.ContentLength only if available and greater than zero)
//
// If onlyHeaders are passed, then only those headers are logged if available.
// If no onlyHeaders are passed, then all headers
// not in the package level FilterHTTPHeaders map will be logged.
//
// To disable header logging, pass an impossible header name
// like an empty string as onlyHeaders.
func (m *Message) Request(request *http.Request, onlyHeaders ...string) *Message {
	if m == nil || request == nil {
		return m
	}

	AttribsFromContext(request.Context()).Log(m)

	m.Str("remote", request.RemoteAddr)
	m.Str("method", request.Method)
	m.Str("uri", request.RequestURI)
	if request.ContentLength > 0 {
		m.Int64("contentLength", request.ContentLength)
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
		return m
	}

	for header, values := range request.Header {
		if _, filter := FilterHTTPHeaders[header]; !filter {
			if len(values) == 1 {
				m.Str(header, values[0])
			} else {
				m.Strs(header, values)
			}
		}
	}
	return m
}

// Log writes the complete log message
// and returns the Message to a memory pool.
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
	m.reset()
	messagePool.PutBack(m)
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
