package golog

import "crypto/rand"

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
