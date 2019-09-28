package golog

import (
	"fmt"
	"sync"
)

type Message struct {
	logger    *Logger
	formatter Formatter
}

var messagePool sync.Pool

func newMessage(logger *Logger, formatter Formatter) *Message {
	if m, ok := messagePool.Get().(*Message); ok {
		m.logger = logger
		m.formatter = formatter
		return m
	}

	return &Message{
		logger:    logger,
		formatter: formatter,
	}
}

func (m *Message) Logger() *Logger {
	return m.logger.newChild(m)
}

// Loggable lets a value that implements the Loggable log itself
func (m *Message) Loggable(key string, val Loggable) *Message {
	if m == nil {
		return nil
	}
	val.LogMessage(m, key)
	return m
}

// Val logs val as string with the "%v" format of the fmt package
func (m *Message) Val(key string, val interface{}) *Message {
	if m == nil {
		return nil
	}
	m.formatter.WriteKey(key)
	m.formatter.WriteString(fmt.Sprint(val))
	return m
}

// Val logs vals as string array with the "%v" format of the fmt package
func (m *Message) Vals(key string, vals []interface{}) *Message {
	if m == nil {
		return nil
	}
	m.formatter.WriteSliceKey(key)
	for _, val := range vals {
		m.formatter.WriteString(fmt.Sprint(val))
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

func (m *Message) Float64(key string, val float64) *Message {
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

func (m *Message) Log() {
	if m == nil {
		return
	}
	m.formatter.FlushAndFree()
	m.formatter = nil
	m.logger = nil
	messagePool.Put(m)
}
