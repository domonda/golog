package golog

// Loggable is an interface that allows types
// to log themselves.
type Loggable interface {
	// LogMessage logs the object to a message.
	LogMessage(message *Message)
}
