package golog

import "time"

// DefaultTimeFormat is the default format used for logging time.Time values
// when Format.TimeFormat is empty.
const DefaultTimeFormat = time.RFC3339Nano

type Format struct {
	TimestampKey    string
	TimestampFormat string
	LevelKey        string // can be empty
	PrefixFmt       string // If there is a prefix then the message will be formatted with `text = fmt.Sprintf(PrefixFmt, prefix, text)`
	MessageKey      string
	TimeFormat      string // Format for time.Time attribute values, defaults to DefaultTimeFormat if empty
}

func NewDefaultFormat() *Format {
	return &Format{
		TimestampKey:    "time",
		TimestampFormat: "2006-01-02 15:04:05.000",
		LevelKey:        "level",
		PrefixFmt:       "%s: %s",
		MessageKey:      "message",
		TimeFormat:      DefaultTimeFormat,
	}
}
