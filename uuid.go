package golog

import (
	"crypto/rand"
	"encoding/hex"
)

// NewUUID returns a new version 4 UUID
func NewUUID() [16]byte {
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

// FormatUUID formats a UUID as string like
// "85692e8d-49bf-4150-a169-6c2adb93463c"
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
