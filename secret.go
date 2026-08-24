package secretsmanager

import "github.com/titpetric/secretsmanager/internal/ulid"

// Secret is one stored secret: the name it's used under, its value, and an
// ID for it.
type Secret struct {
	// ID is a ULID for secrets this tool created. It's kept as a string so
	// the UUIDs written by earlier versions still load.
	ID    string
	Name  string
	Value string

	// raw holds the ciphertext this value was last read or written as, and
	// is empty for a value which hasn't been stored yet. Writing it back
	// verbatim keeps untouched secrets out of the diff, as every encrypt
	// produces a new IV and with it a different ciphertext.
	raw string
}

// secretJSON is one secret as the file holds it, with the value encrypted.
type secretJSON struct {
	ID    string
	Name  string
	Value string
}

// newSecret returns a secret with a new ULID.
func newSecret(name, value string) *Secret {
	return &Secret{
		ID:    ulid.Make(),
		Name:  name,
		Value: value,
	}
}
