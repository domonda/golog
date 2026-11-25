package golog

import (
	"bytes"
	"crypto/rand"
	"encoding"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
)

// UUIDv4 returns a random version 4 UUID
func UUIDv4() [16]byte {
	var id [16]byte
	_, err := rand.Read(id[:])
	if err != nil {
		panic(err)
	}
	// Set version bits
	const version = 4
	id[6] = (id[6] & 0x0f) | (version << 4)
	// Set variant bits as described in RFC 4122
	id[8] = (id[8] & 0xbf) | 0x80
	return id
}

// FormatUUID formats a UUID as 36 character standard string
// looking like "85692e8d-49bf-4150-a169-6c2adb93463c".
func FormatUUID(id [16]byte) string {
	var b [36]byte
	hex.Encode(b[0:8], id[0:4])
	b[8] = '-'
	hex.Encode(b[9:13], id[4:6])
	b[13] = '-'
	hex.Encode(b[14:18], id[6:8])
	b[18] = '-'
	hex.Encode(b[19:23], id[8:10])
	b[23] = '-'
	hex.Encode(b[24:36], id[10:16])
	return string(b[:])
}

// AppendUUID appends the formatted UUID to the byte slice
// and returns the extended slice.
func AppendUUID(buf []byte, id [16]byte) []byte {
	var b [36]byte
	hex.Encode(b[0:8], id[0:4])
	b[8] = '-'
	hex.Encode(b[9:13], id[4:6])
	b[13] = '-'
	hex.Encode(b[14:18], id[6:8])
	b[18] = '-'
	hex.Encode(b[19:23], id[8:10])
	b[23] = '-'
	hex.Encode(b[24:36], id[10:16])
	return append(buf, b[:]...)
}

// MustParseUUID parses a UUID string in the standard
// 36 character format like "85692e8d-49bf-4150-a169-6c2adb93463c"
// and panics on any error.
func MustParseUUID(str string) [16]byte {
	id, err := ParseUUID(str)
	if err != nil {
		panic(err)
	}
	return id
}

// ParseUUID parses a UUID string in the standard
// 36 character format like "85692e8d-49bf-4150-a169-6c2adb93463c".
func ParseUUID(str string) (id [16]byte, err error) {
	if len(str) != 36 {
		return [16]byte{}, fmt.Errorf("invalid UUID string length: %q", str)
	}
	if str[8] != '-' || str[13] != '-' || str[18] != '-' || str[23] != '-' {
		return [16]byte{}, fmt.Errorf("invalid UUID string format: %q", str)
	}

	b := []byte(str)

	_, err = hex.Decode(id[0:4], b[0:8])
	if err != nil {
		return [16]byte{}, fmt.Errorf("error %w parsing UUID string: %q", err, str)
	}
	_, err = hex.Decode(id[4:6], b[9:13])
	if err != nil {
		return [16]byte{}, fmt.Errorf("error %w parsing UUID string: %q", err, str)
	}
	_, err = hex.Decode(id[6:8], b[14:18])
	if err != nil {
		return [16]byte{}, fmt.Errorf("error %w parsing UUID string: %q", err, str)
	}
	_, err = hex.Decode(id[8:10], b[19:23])
	if err != nil {
		return [16]byte{}, fmt.Errorf("error %w parsing UUID string: %q", err, str)
	}
	_, err = hex.Decode(id[10:16], b[24:36])
	if err != nil {
		return [16]byte{}, fmt.Errorf("error %w parsing UUID string: %q", err, str)
	}

	err = ValidateUUID(id)
	if err != nil {
		return [16]byte{}, fmt.Errorf("error %w parsing UUID string: %q", err, str)
	}

	return id, nil
}

// ValidateUUID checks for valid version and variant
// of a binary UUID value.
func ValidateUUID(id [16]byte) error {
	// RFC 9562 defines UUID versions 1 through 8
	// (1-5 from the obsoleted RFC 4122, plus 6, 7 and 8).
	if version := id[6] >> 4; version < 1 || version > 8 {
		return fmt.Errorf("invalid UUID version: %d", version)
	}
	switch {
	case (id[8] & 0x80) == 0x00:
		// Variant NCS
	case (id[8]&0xc0)|0x80 == 0x80:
		// Variant RFC4122
	case (id[8]&0xe0)|0xc0 == 0xc0:
		// Variant Microsoft
	default:
		return errors.New("invalid UUID variant")
	}
	return nil
}

// IsNilUUID checks if the passed id is a Nil UUID
func IsNilUUID(id [16]byte) bool {
	var nilID [16]byte
	return id == nilID
}

func isUUID(v reflect.Value) bool {
	_, ok := asUUID(v)
	return ok
}

// asUUID returns the bytes of v if it is logged as UUID: a [16]byte,
// which Message.Any and Message.UUID log as UUID whatever it holds,
// or a named [16]byte type that holds a valid or nil UUID.
// It copies the bytes with one memmove without allocating,
// because it is called for every array logged with Any.
func asUUID(v reflect.Value) (uuid [16]byte, ok bool) {
	if v.Kind() != reflect.Array || v.Len() != 16 || !v.Type().AssignableTo(uuidType) {
		return [16]byte{}, false
	}
	reflect.Copy(reflect.ValueOf(&uuid).Elem(), v)
	if v.Type() != uuidType && ValidateUUID(uuid) != nil && !IsNilUUID(uuid) {
		return [16]byte{}, false
	}
	return uuid, true
}

var uuidType = reflect.TypeFor[[16]byte]()

// marshalsAsUUID returns true if the MarshalJSON or MarshalText method
// of x, which encoding/json would use, formats uuid like FormatUUID,
// like the UUID type of github.com/google/uuid does, so x is logged as
// UUID, and a nil UUID as null. A type that formats itself differently,
// for example to redact a secret, is logged with its method instead.
func marshalsAsUUID(x any, uuid [16]byte) bool {
	var formatted [38]byte
	switch m := x.(type) {
	case json.Marshaler:
		j, err := safeMarshalJSON(m)
		quoted := append(AppendUUID(append(formatted[:0], '"'), uuid), '"')
		return err == nil && bytes.Equal(j, quoted)
	case encoding.TextMarshaler:
		text, err := safeMarshalText(m)
		return err == nil && bytes.Equal(text, AppendUUID(formatted[:0], uuid))
	}
	return false
}
