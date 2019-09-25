package golog

import (
	"fmt"
	"log"
)

// LevelPrinter is a compatibility shim that enables
// using a Logger with a Level for third party packages
// that either need an an interface implementation with
// a Printf method, an io.Writer, or a standard log.Logger.
type LevelPrinter struct {
	logger *Logger
	level  Level
}

func (p *LevelPrinter) Write(data []byte) (int, error) {
	p.Msg(string(data))
	return len(data), nil
}

func (p *LevelPrinter) Msg(msg string) {
	p.logger.NewMessage(p.level, msg).Log()
}

func (p *LevelPrinter) Print(v ...interface{}) {
	p.Msg(fmt.Sprint(v...))
}

func (p *LevelPrinter) Println(v ...interface{}) {
	p.Msg(fmt.Sprintln(v...))
}

func (p *LevelPrinter) Printf(format string, v ...interface{}) {
	p.logger.NewMessagef(p.level, format, v...).Log()
}

func (p *LevelPrinter) Func() func(format string, v ...interface{}) {
	return func(format string, v ...interface{}) {
		p.Printf(format, v...)
	}
}

func (p *LevelPrinter) StdLogger(prefix string, flag int) *log.Logger {
	return log.New(p, prefix, flag)
}
