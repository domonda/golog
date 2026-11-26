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
//
// The typed Message methods like Str, Int or Strs never call Log.
// Calling one is the caller's choice to log the underlying value under
// the given key, even if the value's type implements Loggable.
// A password type whose Log method redacts the value is logged in plain
// text when it is passed to Str, so pass it to Loggable or Any instead:
//
//	type Password string
//
//	func (Password) Log(m *Message) { m.Str("password", "***") }
//
//	msg.Str("pw", pw)  // pw="secret"
//	msg.Loggable(pw)   // password="***"
//	msg.Any("pw", pw)  // pw={"password":"***"}
type Loggable interface {
	// Log the implementing type to a message
	Log(*Message)
}

// LoggableFunc implements Loggable with a function
type LoggableFunc func(*Message)

func (f LoggableFunc) Log(m *Message) { f(m) }
