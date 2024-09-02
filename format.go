package golog

type Format struct {
	TimestampKey    string
	TimestampFormat string
	LevelKey        string // can be empty
	PrefixFmt       string // If there is a prefix then the message will be formatted with `text = fmt.Sprintf(PrefixFmt, prefix, text)`
	MessageKey      string
}

func NewDefaultFormat() *Format {
	return &Format{
		TimestampKey:    "time",
		TimestampFormat: "2006-01-02 15:04:05.000",
		LevelKey:        "level",
		PrefixFmt:       "%s: %s",
		MessageKey:      "message",
	}
}
