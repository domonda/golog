package golog

import "time"

// DefaultTimeFormat is the layout used for structured [time.Time] log attributes
// ([Message.Time] and related) when [Format.TimeFormat] is empty.
const DefaultTimeFormat = time.RFC3339Nano

// Format configures how writers encode the log line (timestamp, level, message text)
// and how structured time fields are formatted. It is used by [JSONWriterConfig],
// [TextWriterConfig], and related writers; pass nil for format to constructors and they
// substitute [NewDefaultFormat].
//
// Two timestamp-related settings are easy to confuse:
//   - [Format.TimestampFormat] applies to the log record’s event time (the value taken
//     from context for each line). It affects JSON and text output for that leading
//     timestamp only.
//   - [Format.TimeFormat] applies to [time.Time] values logged as message attributes,
//     not to the line timestamp. When empty, [DefaultTimeFormat] is used.
//
// Empty string keys ([Format.TimestampKey], [Format.LevelKey], [Format.MessageKey])
// omit that field in JSON output; see [JSONWriter.BeginMessage].
type Format struct {
	// TimestampKey is the JSON object key for the log line’s event time. Ignored when empty.
	// Text writers do not use this field; they print the timestamp without a key.
	TimestampKey string

	// TimestampFormat is the Go time layout for the log line’s event time in both
	// [JSONWriter] and [TextWriter] ([time.Time.Format] / [time.Time.AppendFormat]).
	// It does not apply to structured [time.Time] attributes; use [Format.TimeFormat] for those.
	TimestampFormat string

	// LevelKey is the JSON object key for the level name. Ignored when empty.
	LevelKey string

	// PrefixFmt is used when the message has a non-empty package prefix: the text becomes
	// fmt.Sprintf(PrefixFmt, prefix, text). The default from [NewDefaultFormat] is "%s: %s".
	PrefixFmt string

	// MessageKey is the JSON object key for the main message string. Ignored when empty
	// or when the message text is empty.
	MessageKey string

	// TimeFormat is the layout for structured [time.Time] fields on the message (e.g.
	// [Message.Time]). If empty, [DefaultTimeFormat] is used. It does not affect the
	// log line timestamp ([Format.TimestampFormat]).
	TimeFormat string

	// Location, when not nil, converts every formatted time value to this
	// location via [time.Time.In] before formatting. It applies to both the
	// log line timestamp ([Format.TimestampFormat]) and structured [time.Time]
	// attributes ([Format.TimeFormat]). When nil, times are formatted in their
	// original location.
	Location *time.Location
}

// NewDefaultFormat returns a pointer to a [Format] with common defaults:
//
//	TimestampKey:    "time"
//	TimestampFormat: "2006-01-02 15:04:05.000"
//	LevelKey:        "level"
//	PrefixFmt:       "%s: %s"
//	MessageKey:      "message"
//	TimeFormat:      [DefaultTimeFormat] (RFC3339Nano)
//	Location:        nil (times keep their original location)
//
// Writers use this when no custom [Format] is supplied. Copy and modify fields to build
// a custom layout (for example a different [Format.TimestampFormat], JSON field names,
// or a [Format.Location] to render every time value in a fixed timezone).
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
