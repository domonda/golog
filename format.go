package golog

type Format struct {
	TimestampKey    string
	TimestampFormat string
	LevelKey        string // can be empty
	PrefixSep       string
	MessageKey      string
}

func NewDefaultFormat() *Format {
	return &Format{
		TimestampKey:    "time",
		TimestampFormat: "2006-01-02 15:04:05.000",
		LevelKey:        "level",
		PrefixSep:       ": ",
		MessageKey:      "message",
	}
}
