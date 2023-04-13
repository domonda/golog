package golog

type Format struct {
	TimestampKey    string
	TimestampFormat string
	LevelKey        string // can be empty
	PrefixSep       string
	MessageKey      string
}
