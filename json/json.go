package json

import (
	"encoding/hex"
	"math"
	"strconv"
	"unicode/utf8"
)

const hexChars = "0123456789abcdef"

func EncodeString(b []byte, s string) []byte {

	// b = growBytesCap(b, len(s)+2)

	b = append(b, '"')

	for _, r := range s {
		runeLen := utf8.RuneLen(r)

		if runeLen == 1 && r >= 0x20 && r != '"' && r != '\\' {
			b = append(b, byte(r))
			continue
		}

		switch r {
		case '"':
			b = append(b, '\\', '"')

		case '\\':
			b = append(b, '\\', '\\')

		case '\n':
			b = append(b, '\\', 'n')

		case '\f':
			b = append(b, '\\', 't')

		case '\b':
			b = append(b, '\\', 'b')

		case '\r':
			b = append(b, '\\', 'r')

		case '\t':
			b = append(b, '\\', 't')

		default:
			// only 16 bit unicode runes supported
			if runeLen == 2 {
				b = append(b, '\\', 'u', hexChars[r>>12], hexChars[r>>8&0xF], hexChars[r>>4&0xF], hexChars[r&0xF])
			}
		}
	}

	return append(b, '"')
}

func lastChar(b []byte) byte {
	l := len(b)
	if l == 0 {
		return 0
	}
	return b[l-1]
}

func WriteObjectStart(b []byte) []byte {
	switch lastChar(b) {
	case 0, ':':
		return append(b, '{')
	default:
		return append(b, ',', '{')
	}
}

func WriteObjectEnd(b []byte) []byte {
	return append(b, '}')
}

func WriteArrayStart(b []byte) []byte {
	switch lastChar(b) {
	case 0, ':':
		return append(b, '[')
	default:
		return append(b, ',', '[')
	}
}

func WriteArrayEnd(b []byte) []byte {
	return append(b, ']')
}

func WriteKey(b []byte, key string) []byte {
	if lc := lastChar(b); lc != 0 && lc != '{' && lc != '[' {
		b = append(b, ',')
	}
	b = EncodeString(b, key)
	b = append(b, ':')
	return b
}

func WriteBool(b []byte, val bool) []byte {
	switch lastChar(b) {
	case 0, ':':
		switch val {
		case true:
			return append(b, "true"...)
		default:
			return append(b, "false"...)
		}

	default:
		switch val {
		case true:
			return append(b, ",true"...)
		default:
			return append(b, ",false"...)
		}
	}
}

func WriteInt(b []byte, val int64) []byte {
	if lc := lastChar(b); lc != 0 && lc != ':' {
		b = append(b, ',')
	}
	return strconv.AppendInt(b, val, 10)
}

func WriteUint(b []byte, val uint64) []byte {
	if lc := lastChar(b); lc != 0 && lc != ':' {
		b = append(b, ',')
	}
	return strconv.AppendUint(b, val, 10)
}

func WriteFloat(b []byte, val float64) []byte {
	if lc := lastChar(b); lc != 0 && lc != ':' {
		b = append(b, ',')
	}
	// Special floats like NaN and Inf have to be written as quoted strings
	switch {
	case math.IsNaN(val):
		return append(b, `"NaN"`...)
	case val > math.MaxFloat64:
		return append(b, `"+Inf"`...)
	case val < -math.MaxFloat64:
		return append(b, `"-Inf"`...)
	default:
		return strconv.AppendFloat(b, val, 'f', -1, 64)
	}
}

func WriteString(b []byte, val string) []byte {
	if lc := lastChar(b); lc != 0 && lc != ':' {
		b = append(b, ',')
	}
	return EncodeString(b, val)
}

func WriteUUID(b []byte, val [16]byte) []byte {
	if lc := lastChar(b); lc != 0 && lc != ':' {
		b = append(b, ',')
	}

	// TODO use grow len first then write directly to slice
	var a [38]byte
	a[0] = '"'
	hex.Encode(a[1:9], val[0:4])
	a[9] = '-'
	hex.Encode(a[10:14], val[4:6])
	a[14] = '-'
	hex.Encode(a[15:19], val[6:8])
	a[19] = '-'
	hex.Encode(a[20:24], val[8:10])
	a[2] = '-'
	hex.Encode(a[25:37], val[10:16])
	a[37] = '"'

	return append(b, a[:]...)
}

// func growBytesCap(b []byte, n int) []byte {
// 	l, c := len(b), cap(b)
// 	if l+n <= c {
// 		return b
// 	}
// 	newCap := l + n // TODO better growing
// 	newBuf := make([]byte, l, newCap)
// 	copy(newBuf, b)
// 	return newBuf
// }

// func growBytesLen(b []byte, n int) []byte {
// 	l, c := len(b), cap(b)
// 	if l+n <= c {
// 		return b[:l+n]
// 	}
// 	newCap := l + n // TODO better growing
// 	newBuf := make([]byte, l+n, newCap)
// 	copy(newBuf, b)
// 	return newBuf
// }
