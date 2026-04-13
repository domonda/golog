package golog

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestNewTimestamp(t *testing.T) {
	before := time.Now()
	ts := TimestampNow()
	after := time.Now()

	if ts.IsNull() {
		t.Error("NewTimestamp() should not be null")
	}
	if ts.Time.Before(before) || ts.Time.After(after) {
		t.Errorf("NewTimestamp() = %v, want in [%v, %v]", ts.Time, before, after)
	}
}

func TestTimestamp_IsNull(t *testing.T) {
	var zero Timestamp
	if !zero.IsNull() {
		t.Error("zero Timestamp: IsNull() = false, want true")
	}

	set := Timestamp{Time: time.Date(2006, 1, 2, 15, 4, 5, 0, time.UTC)}
	if set.IsNull() {
		t.Error("non-zero Timestamp: IsNull() = true, want false")
	}
}

func TestTimestamp_SetNull(t *testing.T) {
	ts := Timestamp{Time: time.Date(2006, 1, 2, 15, 4, 5, 0, time.UTC)}
	ts.SetNull()
	if !ts.IsNull() {
		t.Error("after SetNull(): IsNull() = false, want true")
	}
	if !ts.Time.IsZero() {
		t.Errorf("after SetNull(): Time = %v, want zero", ts.Time)
	}
}

func TestTimestamp_Set(t *testing.T) {
	want := time.Date(2026, 4, 13, 12, 0, 0, 0, time.UTC)
	var ts Timestamp
	ts.Set(want)
	if !ts.Time.Equal(want) {
		t.Errorf("after Set(%v): Time = %v", want, ts.Time)
	}
}

func TestTimestamp_MarshalJSON_Null(t *testing.T) {
	var ts Timestamp
	got, err := json.Marshal(ts)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(got) != "null" {
		t.Errorf("got %s, want null", got)
	}
}

func TestTimestamp_Scan(t *testing.T) {
	rfc := "2006-01-02T15:04:05Z"
	wantRFC, err := ParseTimestamp(rfc)
	if err != nil {
		t.Fatalf("ParseTimestamp(%q): %v", rfc, err)
	}
	fixed := time.Date(2026, 4, 13, 12, 0, 0, 0, time.UTC)

	t.Run("nil SQL NULL clears to zero", func(t *testing.T) {
		ts := Timestamp{Time: fixed}
		if err := ts.Scan(nil); err != nil {
			t.Fatalf("Scan(nil): %v", err)
		}
		if !ts.IsNull() {
			t.Errorf("IsNull() = false, want true (got %v)", ts.Time)
		}
	})

	t.Run("time.Time assigns directly", func(t *testing.T) {
		var ts Timestamp
		if err := ts.Scan(fixed); err != nil {
			t.Fatalf("Scan(time.Time): %v", err)
		}
		if !ts.Time.Equal(fixed) {
			t.Errorf("got %v, want %v", ts.Time, fixed)
		}
	})

	t.Run("int64 Unix epoch seconds", func(t *testing.T) {
		var ts Timestamp
		const sec int64 = 1136239445
		if err := ts.Scan(sec); err != nil {
			t.Fatalf("Scan(int64): %v", err)
		}
		want := time.Unix(sec, 0)
		if !ts.Time.Equal(want) {
			t.Errorf("got %v, want %v", ts.Time, want)
		}
	})

	t.Run("int64 zero is Unix epoch", func(t *testing.T) {
		var ts Timestamp
		if err := ts.Scan(int64(0)); err != nil {
			t.Fatalf("Scan(int64(0)): %v", err)
		}
		if !ts.Time.Equal(time.Unix(0, 0)) {
			t.Errorf("got %v, want Unix epoch", ts.Time)
		}
	})

	t.Run("string uses ParseTimestamp", func(t *testing.T) {
		var ts Timestamp
		if err := ts.Scan(rfc); err != nil {
			t.Fatalf("Scan(string): %v", err)
		}
		if !ts.Time.Equal(wantRFC.Time) {
			t.Errorf("got %v, want %v", ts.Time, wantRFC.Time)
		}
	})

	t.Run("string empty is error from ParseTimestamp", func(t *testing.T) {
		var ts Timestamp
		err := ts.Scan("")
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "golog.Timestamp: Scan:") {
			t.Errorf("error %q should mention Scan", err.Error())
		}
		_, parseErr := ParseTimestamp("")
		if parseErr == nil {
			t.Fatal(`ParseTimestamp("") must fail`)
		}
		if inner := errors.Unwrap(err); inner == nil || inner.Error() != parseErr.Error() {
			t.Errorf("unwrap = %v, want %v", inner, parseErr)
		}
	})

	t.Run("string unparseable returns wrapped error", func(t *testing.T) {
		var ts Timestamp
		err := ts.Scan("2006-01-02")
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "golog.Timestamp: Scan:") {
			t.Errorf("error %q should mention Scan", err.Error())
		}
		_, wantErr := ParseTimestamp("2006-01-02")
		if wantErr == nil {
			t.Fatal("ParseTimestamp should fail for date-only string")
		}
		if inner := errors.Unwrap(err); inner == nil || inner.Error() != wantErr.Error() {
			t.Errorf("Scan error = %v, want unwrap matching ParseTimestamp: %v", err, wantErr)
		}
	})

	t.Run("[]byte uses ParseTimestamp", func(t *testing.T) {
		var ts Timestamp
		if err := ts.Scan([]byte(rfc)); err != nil {
			t.Fatalf("Scan([]byte): %v", err)
		}
		if !ts.Time.Equal(wantRFC.Time) {
			t.Errorf("got %v, want %v", ts.Time, wantRFC.Time)
		}
	})

	t.Run("[]byte empty is error from ParseTimestamp", func(t *testing.T) {
		var ts Timestamp
		err := ts.Scan([]byte{})
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "golog.Timestamp: Scan:") {
			t.Errorf("got %v", err)
		}
	})

	t.Run("nil byte slice is empty string to ParseTimestamp", func(t *testing.T) {
		var ts Timestamp
		var nilBytes []byte
		err := ts.Scan(nilBytes)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("[]byte unparseable returns wrapped error", func(t *testing.T) {
		var ts Timestamp
		err := ts.Scan([]byte("totally invalid"))
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "golog.Timestamp: Scan:") {
			t.Errorf("error %q should mention Scan", err.Error())
		}
	})

	t.Run("unsupported types use default branch", func(t *testing.T) {
		badValues := []any{
			int(42),
			uint64(1),
			float64(1.5),
			true,
			struct{}{},
			[]string{"x"},
		}
		for _, v := range badValues {
			var ts Timestamp
			err := ts.Scan(v)
			if err == nil {
				t.Errorf("Scan(%T) expected error, got nil", v)
				continue
			}
			if !strings.Contains(err.Error(), "failed to scan") {
				t.Errorf("Scan(%T): %v — want failed to scan message", v, err)
			}
		}
	})
}

func TestTimestamp_Scan_nilReceiverPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("expected panic when Scan is called on nil *Timestamp")
		}
	}()
	var ts *Timestamp
	_ = ts.Scan(nil)
}

func TestParseTimestamp_Formats(t *testing.T) {
	ref := time.Date(2006, 1, 2, 15, 4, 5, 123456789, time.UTC)
	for _, layout := range TimestampFormats {
		s := ref.Format(layout)
		want, err := time.Parse(layout, s)
		if err != nil {
			t.Fatalf("time.Parse(%q, formatted): %v", layout, err)
		}
		got, err := ParseTimestamp(s)
		if err != nil {
			t.Fatalf("ParseTimestamp(%q) layout %q: %v", s, layout, err)
		}
		if !got.Time.Equal(want) {
			t.Errorf("layout %q: got %v, want %v", layout, got.Time, want)
		}
	}
}

func TestTimestamp_Value(t *testing.T) {
	var ts Timestamp
	v, err := ts.Value()
	if err != nil {
		t.Fatalf("null Value(): %v", err)
	}
	if v != nil {
		t.Errorf("null Value() = %v, want nil", v)
	}

	want := time.Date(2026, 4, 13, 12, 0, 0, 0, time.UTC)
	ts = Timestamp{Time: want}
	v, err = ts.Value()
	if err != nil {
		t.Fatalf("Value(): %v", err)
	}
	got, ok := v.(time.Time)
	if !ok {
		t.Fatalf("Value() = %T, want time.Time", v)
	}
	if !got.Equal(want) {
		t.Errorf("Value() = %v, want %v", got, want)
	}
}

func TestTimestamp_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  time.Time
	}{
		{
			name:  "RFC3339",
			input: `"2006-01-02T15:04:05Z"`,
			want:  time.Date(2006, 1, 2, 15, 4, 5, 0, time.UTC),
		},
		{
			name:  "RFC3339 with offset",
			input: `"2006-01-02T15:04:05+02:00"`,
			want:  time.Date(2006, 1, 2, 15, 4, 5, 0, time.FixedZone("", 2*60*60)),
		},
		{
			name:  "RFC3339 with millis",
			input: `"2006-01-02T15:04:05.123Z"`,
			want:  time.Date(2006, 1, 2, 15, 4, 5, 123_000_000, time.UTC),
		},
		{
			name:  "RFC3339 with nanos",
			input: `"2006-01-02T15:04:05.123456789Z"`,
			want:  time.Date(2006, 1, 2, 15, 4, 5, 123_456_789, time.UTC),
		},
		{
			name:  "space datetime",
			input: `"2006-01-02 15:04:05"`,
			want:  time.Date(2006, 1, 2, 15, 4, 5, 0, time.UTC),
		},
		{
			name:  "space datetime with millis",
			input: `"2006-01-02 15:04:05.123"`,
			want:  time.Date(2006, 1, 2, 15, 4, 5, 123_000_000, time.UTC),
		},
		{
			name:  "space datetime with micros",
			input: `"2006-01-02 15:04:05.123456"`,
			want:  time.Date(2006, 1, 2, 15, 4, 5, 123_456_000, time.UTC),
		},
		{
			name:  "slash datetime (Go log default)",
			input: `"2006/01/02 15:04:05"`,
			want:  time.Date(2006, 1, 2, 15, 4, 5, 0, time.UTC),
		},
		{
			name:  "Apache / NGINX CLF",
			input: `"02/Jan/2006:15:04:05 -0700"`,
			want:  time.Date(2006, 1, 2, 15, 4, 5, 0, time.FixedZone("", -7*60*60)),
		},
		{
			name:  "RFC1123",
			input: `"Mon, 02 Jan 2006 15:04:05 GMT"`,
			want:  time.Date(2006, 1, 2, 15, 4, 5, 0, time.FixedZone("GMT", 0)),
		},
		{
			name:  "RFC1123Z",
			input: `"Mon, 02 Jan 2006 15:04:05 -0700"`,
			want:  time.Date(2006, 1, 2, 15, 4, 5, 0, time.FixedZone("", -7*60*60)),
		},
		{
			name:  "syslog Stamp",
			input: `"Jan  2 15:04:05"`,
			want:  time.Date(0, 1, 2, 15, 4, 5, 0, time.UTC),
		},
		{
			name:  "syslog StampNano",
			input: `"Jan  2 15:04:05.123456789"`,
			want:  time.Date(0, 1, 2, 15, 4, 5, 123_456_789, time.UTC),
		},
		{
			name:  "compact ISO 8601 basic UTC",
			input: `"20060102T150405Z"`,
			want:  time.Date(2006, 1, 2, 15, 4, 5, 0, time.UTC),
		},
		{
			name:  "compact ISO 8601 basic with offset",
			input: `"20060102T150405+0200"`,
			want:  time.Date(2006, 1, 2, 15, 4, 5, 0, time.FixedZone("", 2*60*60)),
		},
		{
			name:  "compact ISO 8601 basic naive",
			input: `"20060102T150405"`,
			want:  time.Date(2006, 1, 2, 15, 4, 5, 0, time.UTC),
		},
		{
			name:  "unix epoch seconds",
			input: `1136239445`,
			want:  time.Unix(1136239445, 0),
		},
		{
			name:  "null",
			input: `null`,
			want:  time.Time{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got Timestamp
			if err := json.Unmarshal([]byte(tt.input), &got); err != nil {
				t.Fatalf("unmarshal %s: %v", tt.input, err)
			}
			if !got.Equal(tt.want) {
				t.Errorf("got %v, want %v", got.Time, tt.want)
			}
		})
	}
}

func TestTimestamp_MarshalJSON(t *testing.T) {
	ts := Timestamp{Time: time.Date(2006, 1, 2, 15, 4, 5, 123_456_789, time.UTC)}
	got, err := json.Marshal(ts)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	want := `"2006-01-02T15:04:05.123456789Z"`
	if string(got) != want {
		t.Errorf("got %s, want %s", got, want)
	}

	var round Timestamp
	if err := json.Unmarshal(got, &round); err != nil {
		t.Fatalf("roundtrip unmarshal: %v", err)
	}
	if !round.Equal(ts.Time) {
		t.Errorf("roundtrip mismatch: got %v, want %v", round.Time, ts.Time)
	}
}

func TestTimestamp_UnmarshalJSON_Invalid(t *testing.T) {
	inputs := []string{
		`"not a date"`,
		`"2006-13-99"`,
		`"`,
		`[]`,
	}
	for _, in := range inputs {
		var ts Timestamp
		if err := json.Unmarshal([]byte(in), &ts); err == nil {
			t.Errorf("expected error for %q, got nil", in)
		}
	}
}
