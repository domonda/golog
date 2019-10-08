package golog

import (
	"encoding/json"
	"fmt"
	"os"
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
	m.formatter.WriteString(val.Error())
	return m
}

// Val logs val with the best matching typed log method
// or uses Print if none was found.
func (m *Message) Val(key string, val interface{}) *Message {
	if m == nil {
		return nil
	}

	// TODO: detect type and call matching method

	return m.Print(key, val)
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

func (m *Message) Float32(key string, val float32) *Message {
	if m == nil {
		return nil
	}
	m.formatter.WriteKey(key)
	m.formatter.WriteFloat(float64(val))
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
