package golog

import (
	"bytes"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

var (
	_ sql.Scanner   = (*Timestamp)(nil)
	_ driver.Valuer = Timestamp{}
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

// TimestampNow returns a Timestamp wrapping the current time
// as returned by [time.Now].
func TimestampNow() Timestamp {
	return Timestamp{Time: time.Now()}
}

// TimestampFormats are the layouts tried in order by [ParseTimestamp],
// (*Timestamp).UnmarshalJSON for JSON strings, and string/[]byte values
// in (*Timestamp).Scan.
//
// Go's time.Parse silently accepts a fractional second
// suffix (e.g. ".123", ".123456789") after the seconds
// field even when the layout does not mention one, so a
// single layout without fractional seconds covers all
// precisions for that shape.
var TimestampFormats = []string{
	"2006-01-02 15:04:05.000",    // golog standard format returned by NewDefaultFormat
	"2006/01/02 15:04:05",        // Go stdlib log package default
	time.RFC3339Nano,             // 2006-01-02T15:04:05.999999999Z07:00
	"2006-01-02 15:04:05Z07:00",  // 2006-01-02 15:04:05 with timezone
	"2006-01-02 15:04:05",        // 2006-01-02 15:04:05 UTC / no timezone (time.Parse uses UTC for zoneless layouts)
	"02/Jan/2006:15:04:05 -0700", // Apache / NGINX Common Log Format
	time.RFC1123Z,                // Mon, 02 Jan 2006 15:04:05 -0700
	time.RFC1123,                 // Mon, 02 Jan 2006 15:04:05 MST
	time.Stamp,                   // Jan _2 15:04:05        (syslog, no year)
	"20060102T150405Z0700",       // compact ISO 8601 basic with timezone
	"20060102T150405Z",           // compact ISO 8601 basic UTC
	"20060102T150405",            // compact ISO 8601 basic, UTC / no timezone
}

// ParseTimestamp parses s using the layouts in [TimestampFormats] in order.
func ParseTimestamp(s string) (Timestamp, error) {
	if s == "" {
		return Timestamp{}, fmt.Errorf("golog: failed to parse empty timestamp string")
	}
	for _, layout := range TimestampFormats {
		if parsed, err := time.Parse(layout, s); err == nil {
			return Timestamp{Time: parsed}, nil
		}
	}
	return Timestamp{}, fmt.Errorf("golog: failed to parse %q with any layout in golog.TimestampFormats", s)
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
func (t *Timestamp) Set(value time.Time) {
	t.Time = value
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
// A JSON string is parsed with [ParseTimestamp].
func (t *Timestamp) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		t.Time = time.Time{}
		return nil
	}

	if data[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return fmt.Errorf("golog.Timestamp: JSON string: %w", err)
		}
		parsed, err := ParseTimestamp(s)
		if err != nil {
			return err
		}
		*t = parsed
		return nil
	}

	sec, err := strconv.ParseInt(string(data), 10, 64)
	if err != nil {
		return fmt.Errorf("golog.Timestamp: failed to parse %s as Unix epoch seconds: %w", data, err)
	}
	t.Time = time.Unix(sec, 0)
	return nil
}

// Scan implements [sql.Scanner] for database reads. It accepts nil (SQL NULL),
// [time.Time], string, or []byte. String and byte slices are parsed with [ParseTimestamp].
func (t *Timestamp) Scan(value any) error {
	switch v := value.(type) {
	case nil:
		*t = Timestamp{}
		return nil

	case time.Time:
		t.Time = v
		return nil

	case int64:
		t.Time = time.Unix(v, 0)
		return nil

	case string:
		parsed, err := ParseTimestamp(v)
		if err != nil {
			return fmt.Errorf("golog.Timestamp: Scan: %w", err)
		}
		*t = parsed
		return nil

	case []byte:
		parsed, err := ParseTimestamp(string(v))
		if err != nil {
			return fmt.Errorf("golog.Timestamp: Scan: %w", err)
		}
		*t = parsed
		return nil

	default:
		return fmt.Errorf("golog.Timestamp: failed to scan %T", v)
	}
}

// Value implements the driver database/sql/driver.Valuer interface.
func (t Timestamp) Value() (driver.Value, error) {
	if t.IsNull() {
		return nil, nil
	}
	return t.Time, nil
}
