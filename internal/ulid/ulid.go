// Package ulid makes ULIDs: 128 bit identifiers which sort by the time
// they were made.
//
// A ULID is a 48 bit millisecond timestamp followed by 80 random bits,
// written as 26 characters of Crockford base32. Sorting the text sorts by
// time, which is the property the secrets file wants from an ID.
//
// See https://github.com/ulid/spec.
package ulid

import (
	"crypto/rand"
	"time"
)

// Length is the number of characters a ULID is written as.
const Length = 26

// alphabet is Crockford base32: the digits and the uppercase letters,
// without I, L, O and U, which are the ones misread.
const alphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

// Make returns a ULID for the current time.
func Make() string {
	return New(time.Now())
}

// New returns a ULID for a time.
func New(t time.Time) string {
	var id [16]byte

	// The timestamp is the top 48 bits, most significant byte first, so
	// that ordering the bytes orders the times.
	ms := uint64(t.UnixMilli())
	id[0] = byte(ms >> 40)
	id[1] = byte(ms >> 32)
	id[2] = byte(ms >> 24)
	id[3] = byte(ms >> 16)
	id[4] = byte(ms >> 8)
	id[5] = byte(ms)

	// The remaining 80 bits are entropy. crypto/rand.Read fills the slice
	// or panics; it doesn't report a failure to its caller.
	rand.Read(id[6:])

	return encode(id)
}

// Valid reports whether id is written as a ULID. It doesn't say the ULID
// was made here, only that it could have been.
func Valid(id string) bool {
	if len(id) != Length {
		return false
	}

	for i := range id {
		if index(id[i]) < 0 {
			return false
		}
	}

	// The first character carries the top 3 bits of a 128 bit value spread
	// over 130, so anything past 7 doesn't fit.
	return id[0] <= '7'
}

// encode writes the 128 bits as 26 base32 characters, most significant
// first. The first character holds 3 bits, the other 25 hold 5 each.
func encode(id [16]byte) string {
	var hi, lo uint64
	for i := range 8 {
		hi = hi<<8 | uint64(id[i])
		lo = lo<<8 | uint64(id[8+i])
	}

	out := make([]byte, Length)
	for i := range out {
		out[i] = alphabet[fiveBits(hi, lo, uint(5*(Length-1-i)))]
	}

	return string(out)
}

// fiveBits returns the five bits of a 128 bit value at an offset, counted
// from the least significant bit.
func fiveBits(hi, lo uint64, shift uint) byte {
	if shift >= 64 {
		return byte(hi>>(shift-64)) & 0x1f
	}

	// Shifting a uint64 by 64 gives zero, which is what a shift of 0 wants
	// from the high half.
	return byte(lo>>shift|hi<<(64-shift)) & 0x1f
}

// index returns the value of a base32 character, or -1.
func index(c byte) int {
	for i := range len(alphabet) {
		if alphabet[i] == c {
			return i
		}
	}
	return -1
}
