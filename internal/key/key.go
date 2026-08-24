// Package key makes and reads the encryption key the secrets are stored
// under.
package key

import (
	"crypto/rand"
	"errors"
	"fmt"
	"os"
)

// Length is the number of characters a generated key has, and the only
// length read from the environment. AES also takes 16 and 24 byte keys,
// which a caller may configure directly.
const Length = 32

// alphabet is what a generated key is made of: the digits and letters,
// without 0, 1, i, I, l, L, o and O, which are the ones misread off a
// screen or a printout.
const alphabet = "23456789ABCDEFGHJKMNPQRSTUVWXYZabcdefghjkmnpqrstuvwxyz"

// Generate returns a new encryption key, which is what init prints.
func Generate() string {
	// Values at the end of the byte range which don't complete a run of the
	// alphabet are drawn again, so no character comes up more often than
	// the others.
	limit := 256 - (256 % len(alphabet))

	key := make([]byte, Length)
	buf := make([]byte, 1)

	for i := range key {
		for {
			// crypto/rand.Read fills the slice or panics; it doesn't report
			// a failure to its caller.
			rand.Read(buf)
			if int(buf[0]) < limit {
				key[i] = alphabet[int(buf[0])%len(alphabet)]
				break
			}
		}
	}

	return string(key)
}

// FromEnv returns the encryption key held in SECRETSMANAGER_KEY.
func FromEnv() ([]byte, error) {
	key := os.Getenv("SECRETSMANAGER_KEY")
	switch {
	case key == "":
		return nil, errors.New("SECRETSMANAGER_KEY is not set, generate one with 'secretsmanager init'")
	case len(key) != Length:
		return nil, fmt.Errorf("SECRETSMANAGER_KEY must be %d characters, got %d", Length, len(key))
	}

	return []byte(key), nil
}
