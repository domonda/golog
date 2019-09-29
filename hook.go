package golog

type Hook interface {
	Log(*Message)
}

type HookFunc func(*Message)

func (f HookFunc) Log(message *Message) { f(message) }
