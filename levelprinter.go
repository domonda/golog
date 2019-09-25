package golog

// type Printer interface {
// 	Printf(format string, v ...interface{})
// }

// LevelPrinter is a compatibility shim to be able to use
// a Logger with a Level for third party packages
// that need an an interface implementation with
// a Printf(string, ...interface{}) method.
type LevelPrinter struct {
	logger *Logger
	level  Level
}

func (p *LevelPrinter) Printf(format string, v ...interface{}) {
	p.logger.NewMessagef(p.level, format, v...).Log()
}

func (p *LevelPrinter) Func() func(format string, v ...interface{}) {
	return func(format string, v ...interface{}) {
		p.Printf(format, v...)
	}
}
