package secretsmanager

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"

	"github.com/titpetric/secretsmanager/internal/envname"
	"github.com/titpetric/secretsmanager/internal/key"
)

// fileStorage keeps the secrets in a JSON file. It's the driver behind a
// workspace which is a directory.
//
// The file is read on the first call which needs it, and written by Set.
// Reading a secret never touches it, which matters because every encrypt
// produces a new IV: rewriting a file that didn't change would rewrite
// every value in it.
type fileStorage struct {
	filename string
	key      []byte
	loaded   bool
	secrets  []*Secret
}

// fileDocument is the shape of the secrets file.
type fileDocument struct {
	Secrets []secretJSON `json:"secrets,omitempty"`
}

// fileStorage is the only driver so far.
var _ Storage = (*fileStorage)(nil)

// newFileStorage returns the driver for a secrets file, holding the key its
// values are encrypted with. Each storage keeps its own, so two of them can
// serve two workspaces at once.
//
// A nil key is read from SECRETSMANAGER_KEY when the file is first used,
// which is what the cli does: init has to run before there is a key at all.
func newFileStorage(filename string, key []byte) *fileStorage {
	return &fileStorage{
		filename: filename,
		key:      key,
	}
}

// String returns the path of the secrets file.
func (s *fileStorage) String() string {
	return s.filename
}

// List returns the secrets in the order they're stored in the file.
func (s *fileStorage) List(ctx context.Context) ([]*Secret, error) {
	if err := s.load(ctx); err != nil {
		return nil, err
	}
	return s.secrets, nil
}

// Get returns the secret stored under a name. Secrets are matched on the
// environment name they produce, as that's the name they're used under, and
// two secrets sharing one would shadow each other.
func (s *fileStorage) Get(ctx context.Context, name string) (*Secret, error) {
	if err := s.load(ctx); err != nil {
		return nil, err
	}

	key := envname.Name(name)

	for _, secret := range s.secrets {
		if envname.Name(secret.Name) == key {
			return secret, nil
		}
	}
	return nil, fmt.Errorf("%w: %q in %s", ErrNotFound, name, s.filename)
}

// Set adds a secret, or replaces the value of an existing one, keeping its
// ID, its name and its position in the file. The file is written before it
// returns, and only the secret which changed is encrypted again.
func (s *fileStorage) Set(ctx context.Context, name, value string) (*Secret, error) {
	secret, err := s.Get(ctx, name)
	switch {
	case err == nil:
		// Encrypting the same value again would only give it a new IV, and
		// with it a diff on a secret nobody changed.
		if secret.Value == value {
			return secret, nil
		}

		secret.Value = value
		secret.raw = ""
	case errors.Is(err, ErrNotFound):
		secret = newSecret(name, value)
		s.secrets = append(s.secrets, secret)
	default:
		return nil, err
	}

	if err := s.save(ctx); err != nil {
		return nil, err
	}
	return secret, nil
}

// cryptKey returns the key the values are encrypted with, reading it from
// the environment for a storage which was created without one.
func (s *fileStorage) cryptKey() ([]byte, error) {
	if s.key != nil {
		return s.key, nil
	}

	configured, err := key.FromEnv()
	if err != nil {
		return nil, err
	}

	s.key = configured
	return configured, nil
}

// load reads the secrets file, once. A missing or empty file is an empty
// store, so the first secret can be created without preparing it by hand.
func (s *fileStorage) load(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s.loaded {
		return nil
	}

	key, err := s.cryptKey()
	if err != nil {
		return err
	}

	f, err := os.Open(s.filename)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			s.loaded = true
			return nil
		}
		return err
	}
	defer f.Close()

	var document fileDocument
	if err := json.NewDecoder(f).Decode(&document); err != nil && !errors.Is(err, io.EOF) {
		return err
	}

	// Two secrets sharing an environment name shadow each other: Get and Set
	// would only ever reach the first, while env prints both and whatever
	// reads the result keeps the last. The file has to be fixed by hand.
	stored := make(map[string]string, len(document.Secrets))
	secrets := make([]*Secret, 0, len(document.Secrets))

	for _, entry := range document.Secrets {
		name := envname.Name(entry.Name)
		if first, ok := stored[name]; ok {
			return fmt.Errorf("%s: %q and %q are both stored as %s", s.filename, first, entry.Name, name)
		}
		stored[name] = entry.Name

		value, err := decrypt(key, entry.Value)
		if err != nil {
			return fmt.Errorf("secret %q: %w", entry.Name, err)
		}

		secrets = append(secrets, &Secret{
			ID:    entry.ID,
			Name:  entry.Name,
			Value: value,
			raw:   entry.Value,
		})
	}

	s.secrets = secrets
	s.loaded = true
	return nil
}

// save writes the secrets file, by replacing it with a complete one. Only
// the secrets without a ciphertext to write back are encrypted.
func (s *fileStorage) save(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	key, err := s.cryptKey()
	if err != nil {
		return err
	}

	document := fileDocument{Secrets: make([]secretJSON, 0, len(s.secrets))}
	for _, secret := range s.secrets {
		if secret.raw == "" {
			if secret.raw, err = encrypt(key, secret.Value); err != nil {
				return err
			}
		}

		document.Secrets = append(document.Secrets, secretJSON{
			ID:    secret.ID,
			Name:  secret.Name,
			Value: secret.raw,
		})
	}

	tmpFilename := s.filename + ".tmp"
	f, err := os.OpenFile(tmpFilename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(document); err != nil {
		f.Close()
		os.Remove(tmpFilename)
		return err
	}

	// Sync before the rename, so a crash leaves either the old file or the
	// new one, not an empty one.
	if err := f.Sync(); err != nil {
		f.Close()
		os.Remove(tmpFilename)
		return err
	}

	if err := f.Close(); err != nil {
		os.Remove(tmpFilename)
		return err
	}

	if err := os.Rename(tmpFilename, s.filename); err != nil {
		os.Remove(tmpFilename)
		return err
	}
	return nil
}
