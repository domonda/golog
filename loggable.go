package golog

// Loggable can be implemented to allow a type to log itself
type Loggable interface {
	// Log the implementing type to a message
	Log(*Message)
}

// LoggableFunc implements Loggable with a function
type LoggableFunc func(*Message)

func (f LoggableFunc) Log(m *Message) { f(m) }
