package golog

import (
	"bytes"
	"database/sql/driver"
	"fmt"
	"strconv"
	"time"
)

// Timestamp wraps a [time.Time] with JSON, SQL and nullability
// helpers tailored for log timestamps.
//
// JSON unmarshalling accepts a wide range of common log timestamp
// string formats (see [TimestampFormats]) as well as a JSON number
// interpreted as Unix epoch seconds. JSON null is treated as the
// zero value. JSON marshalling produces a JSON string formatted
// with [DefaultTimeFormat], or JSON null if the Timestamp is null.
//
// A Timestamp is considered null when the underlying time.Time is
// zero. Timestamp implements [database/sql.Scanner] and
// [database/sql/driver.Valuer] so it can be used directly with
// database/sql, mapping SQL NULL to the zero value.
type Timestamp struct {
	time.Time
}

// NewTimestamp returns a Timestamp wrapping the current time
// as returned by [time.Now].
func NewTimestamp() Timestamp {
	return Timestamp{Time: time.Now()}
}

// TimestampFormats are the layouts tried in order by
// [Timestamp.UnmarshalJSON] when parsing a JSON string.
//
// Go's time.Parse silently accepts a fractional second
// suffix (e.g. ".123", ".123456789") after the seconds
// field even when the layout does not mention one, so a
// single layout without fractional seconds covers all
// precisions for that shape.
var TimestampFormats = []string{
	time.RFC3339Nano,             // 2006-01-02T15:04:05.999999999Z07:00
	"2006-01-02 15:04:05Z07:00",  // 2006-01-02 15:04:05 with timezone
	"2006-01-02 15:04:05",        // 2006-01-02 15:04:05 local / no timezone
	"2006/01/02 15:04:05",        // Go stdlib log package default
	"02/Jan/2006:15:04:05 -0700", // Apache / NGINX Common Log Format
	time.RFC1123Z,                // Mon, 02 Jan 2006 15:04:05 -0700
	time.RFC1123,                 // Mon, 02 Jan 2006 15:04:05 MST
	time.Stamp,                   // Jan _2 15:04:05        (syslog, no year)
	"20060102T150405Z0700",       // compact ISO 8601 basic with timezone
	"20060102T150405Z",           // compact ISO 8601 basic UTC
	"20060102T150405",            // compact ISO 8601 basic, no timezone
}

// IsNull returns true if the Timestamp is null,
// using [time.Time.IsZero] internally.
func (t Timestamp) IsNull() bool {
	return t.Time.IsZero()
}

// SetNull sets the Timestamp to its zero value.
func (t *Timestamp) SetNull() {
	t.Time = time.Time{}
}

// Set overwrites the wrapped time.Time.
func (t *Timestamp) Set(time time.Time) {
	t.Time = time
}

// MarshalJSON formats the Timestamp as a JSON string
// using [DefaultTimeFormat], or as JSON null if [Timestamp.IsNull].
func (t Timestamp) MarshalJSON() ([]byte, error) {
	if t.IsNull() {
		return []byte("null"), nil
	}
	buf := make([]byte, 0, len(DefaultTimeFormat)+2)
	buf = append(buf, '"')
	buf = t.AppendFormat(buf, DefaultTimeFormat)
	buf = append(buf, '"')
	return buf, nil
}

// UnmarshalJSON parses a JSON value into the Timestamp.
// A JSON null leaves the Timestamp zero.
// A JSON number is interpreted as Unix epoch seconds.
// A JSON string is parsed by trying each layout in
// [TimestampFormats] until one succeeds.
func (t *Timestamp) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		t.Time = time.Time{}
		return nil
	}

	if data[0] != '"' {
		sec, err := strconv.ParseInt(string(data), 10, 64)
		if err != nil {
			return fmt.Errorf("golog.Timestamp: cannot parse %s as Unix epoch seconds: %w", data, err)
		}
		t.Time = time.Unix(sec, 0)
		return nil
	}

	if len(data) < 2 || data[len(data)-1] != '"' {
		return fmt.Errorf("golog.Timestamp: invalid JSON string %s", data)
	}
	s := string(data[1 : len(data)-1])

	for _, layout := range TimestampFormats {
		if parsed, err := time.Parse(layout, s); err == nil {
			t.Time = parsed
			return nil
		}
	}
	return fmt.Errorf("golog.Timestamp: cannot parse %q as timestamp", s)
}

// Scan implements the database/sql.Scanner interface.
func (t *Timestamp) Scan(value any) error {
	switch v := value.(type) {
	case nil:
		*t = Timestamp{}
		return nil

	case time.Time:
		t.Time = v
		return nil

	default:
		return fmt.Errorf("can't scan %T as golog.Timestamp", value)
	}
}

// Value implements the driver database/sql/driver.Valuer interface.
func (t Timestamp) Value() (driver.Value, error) {
	if t.IsNull() {
		return nil, nil
	}
	return t.Time, nil
}
