package golog

// Loggable can be implemented to allow a type to log itself.
//
// Message.Loggable logs the attribs of a Loggable as attribs of the message,
// Message.Any logs them as JSON object under a key.
//
// Declare Log with a value receiver to log values of the type.
// A Log method declared on the pointer type is only called for pointers:
// Message.Any encodes a value of such a type with encoding/json instead,
// which logs all exported fields, including those that Log leaves out.
type Loggable interface {
	// Log the implementing type to a message
	Log(*Message)
}

// LoggableFunc implements Loggable with a function
type LoggableFunc func(*Message)

func (f LoggableFunc) Log(m *Message) { f(m) }
