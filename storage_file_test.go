package secretsmanager

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// newTestStorage returns a storage backed by a file in a temporary directory,
// with the encryption key set for the duration of the test.
func newTestStorage(t *testing.T) *fileStorage {
	t.Helper()
	t.Setenv("SECRETSMANAGER_KEY", testKey)

	return newFileStorage(filepath.Join(t.TempDir(), secretsFilename), []byte(testKey))
}

// seed stores the given name and value pairs, and returns the contents of
// the file they were written to.
func seed(t *testing.T, storage *fileStorage, secrets ...string) []byte {
	t.Helper()
	ctx := t.Context()

	for i := 0; i < len(secrets); i += 2 {
		if _, err := storage.Set(ctx, secrets[i], secrets[i+1]); err != nil {
			t.Fatalf("Set(%q): %v", secrets[i], err)
		}
	}

	contents, err := os.ReadFile(storage.filename)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	return contents
}

// list returns the stored secrets, failing the test if they can't be read.
func list(t *testing.T, storage *fileStorage) []*Secret {
	t.Helper()

	secrets, err := storage.List(t.Context())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	return secrets
}

// values decodes a secrets file without decrypting it, so the ciphertext of
// each secret can be compared.
func values(t *testing.T, contents []byte) map[string]string {
	t.Helper()

	var decoded fileDocument
	if err := json.Unmarshal(contents, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	out := make(map[string]string, len(decoded.Secrets))
	for _, secret := range decoded.Secrets {
		out[secret.Name] = secret.Value
	}
	return out
}

// TestFileStorage covers the driver behind a workspace which is a directory.
func TestFileStorage(t *testing.T) {
	t.Run("reading a missing file", func(t *testing.T) {
		storage := newTestStorage(t)

		if secrets := list(t, storage); len(secrets) != 0 {
			t.Errorf("List = %d secrets, want 0", len(secrets))
		}
		if _, err := storage.Get(t.Context(), "DB_DSN"); !errors.Is(err, ErrNotFound) {
			t.Errorf("Get: %v, want %v", err, ErrNotFound)
		}
		if _, err := os.Stat(storage.filename); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("Stat error = %v, want %v", err, fs.ErrNotExist)
		}
	})

	t.Run("reading an empty file", func(t *testing.T) {
		storage := newTestStorage(t)

		if err := os.WriteFile(storage.filename, nil, 0o600); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
		if secrets := list(t, storage); len(secrets) != 0 {
			t.Errorf("List = %d secrets, want 0", len(secrets))
		}
	})

	// Two secrets sharing an environment name shadow each other, so the file
	// is refused rather than half honoured.
	t.Run("reading a file with a name stored twice", func(t *testing.T) {
		storage := newTestStorage(t)

		value, err := encrypt([]byte(testKey), "user:password@hostname")
		if err != nil {
			t.Fatalf("encrypt: %v", err)
		}

		contents := fmt.Sprintf(
			`{"secrets":[{"ID":"1","Name":"DB_DSN","Value":%q},{"ID":"2","Name":"db-dsn","Value":%q}]}`,
			value, value,
		)
		if err := os.WriteFile(storage.filename, []byte(contents), 0o600); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}

		reader := newFileStorage(storage.filename, []byte(testKey))
		_, err = reader.List(t.Context())
		if err == nil {
			t.Fatal("List: expected an error for a name which is stored twice")
		}
		for _, want := range []string{"db-dsn", "DB_DSN"} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("List: %v, want the error to name %q", err, want)
			}
		}
	})

	t.Run("reading a damaged file", func(t *testing.T) {
		storage := newTestStorage(t)

		if err := os.WriteFile(storage.filename, []byte("{"), 0o600); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
		if _, err := storage.List(t.Context()); err == nil {
			t.Error("List: expected an error for a file which isn't JSON")
		}
	})

	// The values aren't authenticated, so the wrong key doesn't fail the
	// read, it gives the wrong value back.
	t.Run("wrong key", func(t *testing.T) {
		storage := newTestStorage(t)
		seed(t, storage, "DB_DSN", "user:password@hostname")

		reader := newFileStorage(storage.filename, []byte(strings.ToUpper(testKey)))

		secrets, err := reader.List(t.Context())
		if err == nil && secrets[0].Value == "user:password@hostname" {
			t.Error("List: a value encrypted with another key was read back intact")
		}
	})

	// A storage built without a key takes one from the environment, which
	// is how the cli works: init has to run before there is a key at all.
	t.Run("no key", func(t *testing.T) {
		filename := filepath.Join(t.TempDir(), secretsFilename)

		t.Setenv("SECRETSMANAGER_KEY", "")
		if _, err := newFileStorage(filename, nil).List(t.Context()); err == nil {
			t.Error("List: expected an error for a missing SECRETSMANAGER_KEY")
		}

		t.Setenv("SECRETSMANAGER_KEY", testKey)
		storage := newFileStorage(filename, nil)
		if _, err := storage.Set(t.Context(), "DB_DSN", "user:password@hostname"); err != nil {
			t.Fatalf("Set: %v", err)
		}
		if got := list(t, storage)[0].Value; got != "user:password@hostname" {
			t.Errorf("Value = %q, want the value it was given", got)
		}
	})

	// Each storage encrypts with the key it was built with. Sharing one
	// through the package would mean the last one built decided for all of
	// them, and quietly wrote a file its owner couldn't read back.
	t.Run("two storages keep their own keys", func(t *testing.T) {
		ctx := t.Context()
		first := newFileStorage(filepath.Join(t.TempDir(), secretsFilename), []byte(testKey))
		second := newFileStorage(filepath.Join(t.TempDir(), secretsFilename), []byte(strings.ToUpper(testKey)))

		if _, err := first.Set(ctx, "DB_DSN", "first value"); err != nil {
			t.Fatalf("Set: %v", err)
		}
		if _, err := second.Set(ctx, "DB_DSN", "second value"); err != nil {
			t.Fatalf("Set: %v", err)
		}
		if _, err := first.Set(ctx, "DB_DSN", "third value"); err != nil {
			t.Fatalf("Set: %v", err)
		}

		reread := newFileStorage(first.filename, []byte(testKey))
		secret, err := reread.Get(ctx, "DB_DSN")
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		if got, want := secret.Value, "third value"; got != want {
			t.Errorf("Value = %q, want %q", got, want)
		}
	})

	// A caller holds a copy. Writing to it would otherwise reach the file
	// the next time anything else was stored.
	t.Run("a secret handed out is a copy", func(t *testing.T) {
		ctx := t.Context()
		storage := newTestStorage(t)
		seed(t, storage, "DB_DSN", "user:password@hostname", "API_KEY", "abc123")

		secret, err := storage.Get(ctx, "DB_DSN")
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		secret.Value = "written through the copy"

		listed := list(t, storage)
		listed[1].Value = "written through the list"
		listed = append(listed, newSecret("EXTRA", "value"))

		if _, err := storage.Set(ctx, "THIRD", "third value"); err != nil {
			t.Fatalf("Set: %v", err)
		}

		stored := list(t, newFileStorage(storage.filename, []byte(testKey)))
		if len(stored) != 3 {
			t.Fatalf("List = %d secrets, want 3", len(stored))
		}
		if got, want := stored[0].Value, "user:password@hostname"; got != want {
			t.Errorf("DB_DSN = %q, want %q", got, want)
		}
		if got, want := stored[1].Value, "abc123"; got != want {
			t.Errorf("API_KEY = %q, want %q", got, want)
		}
	})

	t.Run("reading does not write", func(t *testing.T) {
		ctx := t.Context()
		storage := newTestStorage(t)
		before := seed(t, storage, "DB_DSN", "user:password@hostname")

		reader := newFileStorage(storage.filename, []byte(testKey))
		if secrets := list(t, reader); secrets[0].Value != "user:password@hostname" {
			t.Errorf("Value = %q, want %q", secrets[0].Value, "user:password@hostname")
		}
		if _, err := reader.Get(ctx, "DB_DSN"); err != nil {
			t.Fatalf("Get: %v", err)
		}

		after, err := os.ReadFile(storage.filename)
		if err != nil {
			t.Fatalf("ReadFile: %v", err)
		}
		if string(before) != string(after) {
			t.Errorf("file was rewritten:\nbefore: %s\nafter:  %s", before, after)
		}
	})

	t.Run("set adds only the new secret", func(t *testing.T) {
		storage := newTestStorage(t)
		before := seed(t, storage, "DB_DSN", "user:password@hostname")

		writer := newFileStorage(storage.filename, []byte(testKey))
		if _, err := writer.Set(t.Context(), "API_KEY", "abc123"); err != nil {
			t.Fatalf("Set: %v", err)
		}

		after, err := os.ReadFile(storage.filename)
		if err != nil {
			t.Fatalf("ReadFile: %v", err)
		}

		old, current := values(t, before), values(t, after)
		if len(current) != len(old)+1 {
			t.Fatalf("secrets = %d, want %d", len(current), len(old)+1)
		}
		if current["DB_DSN"] != old["DB_DSN"] {
			t.Errorf("DB_DSN ciphertext changed:\nbefore: %s\nafter:  %s", old["DB_DSN"], current["DB_DSN"])
		}
		if current["API_KEY"] == "" {
			t.Error("API_KEY was not written")
		}
	})

	t.Run("set updates in place", func(t *testing.T) {
		ctx := t.Context()
		storage := newTestStorage(t)
		before := seed(t, storage, "DB_DSN", "user:password@hostname", "API_KEY", "abc123")

		id := list(t, storage)[0].ID
		secret, err := storage.Set(ctx, "DB_DSN", "newvalue")
		if err != nil {
			t.Fatalf("Set: %v", err)
		}
		if secret.ID != id {
			t.Errorf("ID = %s, want %s", secret.ID, id)
		}

		secrets := list(t, storage)
		if len(secrets) != 2 {
			t.Fatalf("List = %d secrets, want 2", len(secrets))
		}
		if secrets[0].ID != secret.ID {
			t.Error("Set: existing secret moved from its position")
		}

		after, err := os.ReadFile(storage.filename)
		if err != nil {
			t.Fatalf("ReadFile: %v", err)
		}
		old, current := values(t, before), values(t, after)
		if current["DB_DSN"] == old["DB_DSN"] {
			t.Error("DB_DSN ciphertext was not rewritten")
		}
		if current["API_KEY"] != old["API_KEY"] {
			t.Error("API_KEY ciphertext changed, only DB_DSN was set")
		}

		stored := list(t, newFileStorage(storage.filename, []byte(testKey)))
		if got, want := stored[0].Value, "newvalue"; got != want {
			t.Errorf("Value = %q, want %q", got, want)
		}
		if got, want := stored[1].Value, "abc123"; got != want {
			t.Errorf("Value = %q, want %q", got, want)
		}
	})

	// Storing the value a secret already holds would encrypt it again, and
	// give it a new IV, so it leaves the file alone.
	t.Run("set of an unchanged value writes nothing", func(t *testing.T) {
		storage := newTestStorage(t)
		before := seed(t, storage, "DB_DSN", "user:password@hostname")

		if _, err := storage.Set(t.Context(), "db-dsn", "user:password@hostname"); err != nil {
			t.Fatalf("Set: %v", err)
		}

		after, err := os.ReadFile(storage.filename)
		if err != nil {
			t.Fatalf("ReadFile: %v", err)
		}
		if string(before) != string(after) {
			t.Errorf("file was rewritten:\nbefore: %s\nafter:  %s", before, after)
		}
	})

	// Names which differ only in case or in their separators share one
	// environment name, so they have to share one entry rather than shadow
	// each other.
	t.Run("set matches on the environment name", func(t *testing.T) {
		storage := newTestStorage(t)
		seed(t, storage, "DB_DSN", "user:password@hostname")

		secret, err := storage.Set(t.Context(), "db-dsn", "newvalue")
		if err != nil {
			t.Fatalf("Set: %v", err)
		}
		if secrets := list(t, storage); len(secrets) != 1 {
			t.Fatalf("List = %d secrets, want 1", len(secrets))
		}
		if got, want := secret.Name, "DB_DSN"; got != want {
			t.Errorf("Name = %q, want %q (the stored name is kept)", got, want)
		}
		if got, want := secret.Value, "newvalue"; got != want {
			t.Errorf("Value = %q, want %q", got, want)
		}
	})

	t.Run("get", func(t *testing.T) {
		ctx := t.Context()
		storage := newTestStorage(t)
		seed(t, storage, "DB_DSN", "user:password@hostname")

		for _, name := range []string{"DB_DSN", "db_dsn", "db-dsn", "dbDsn"} {
			secret, err := storage.Get(ctx, name)
			if err != nil {
				t.Errorf("Get(%q): %v", name, err)
				continue
			}
			if got, want := secret.Value, "user:password@hostname"; got != want {
				t.Errorf("Get(%q) = %q, want %q", name, got, want)
			}
		}

		_, err := storage.Get(ctx, "MISSING")
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("Get: %v, want %v", err, ErrNotFound)
		}
		if !strings.Contains(err.Error(), storage.filename) {
			t.Errorf("Get: %v, want the error to name %s", err, storage.filename)
		}
	})

	t.Run("list keeps the stored order", func(t *testing.T) {
		storage := newTestStorage(t)
		seed(t, storage, "DB_DSN", "user:password@hostname", "API_KEY", "abc123")

		secrets := list(t, storage)
		if len(secrets) != 2 {
			t.Fatalf("List = %d secrets, want 2", len(secrets))
		}
		if got, want := secrets[0].Name, "DB_DSN"; got != want {
			t.Errorf("List[0] = %q, want %q", got, want)
		}
	})

	t.Run("file permissions", func(t *testing.T) {
		storage := newTestStorage(t)
		seed(t, storage, "DB_DSN", "user:password@hostname")

		info, err := os.Stat(storage.filename)
		if err != nil {
			t.Fatalf("Stat: %v", err)
		}
		if got, want := info.Mode().Perm(), fs.FileMode(0o600); got != want {
			t.Errorf("mode = %o, want %o", got, want)
		}
	})

	t.Run("an unwritable file is an error", func(t *testing.T) {
		storage := newTestStorage(t)

		// A directory in place of the temporary file makes the write fail.
		if err := os.Mkdir(storage.filename+".tmp", 0o700); err != nil {
			t.Fatalf("Mkdir: %v", err)
		}
		if _, err := storage.Set(t.Context(), "DB_DSN", "user:password@hostname"); err == nil {
			t.Error("Set: expected an error when the file can't be written")
		}
	})

	// The write lands on a temporary file which is renamed over the real
	// one. A directory in the way of that rename fails the last step.
	t.Run("a failed rename leaves no temporary file", func(t *testing.T) {
		ctx := t.Context()
		storage := newTestStorage(t)

		if _, err := storage.List(ctx); err != nil {
			t.Fatalf("List: %v", err)
		}
		if err := os.Mkdir(storage.filename, 0o700); err != nil {
			t.Fatalf("Mkdir: %v", err)
		}

		if _, err := storage.Set(ctx, "DB_DSN", "user:password@hostname"); err == nil {
			t.Error("Set: expected an error when the file can't be replaced")
		}
		if _, err := os.Stat(storage.filename + ".tmp"); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("Stat error = %v, want the temporary file to be removed", err)
		}
	})

	// Anything other than a missing file is reported, rather than read as an
	// empty store.
	t.Run("a workspace which isn't a directory", func(t *testing.T) {
		t.Setenv("SECRETSMANAGER_KEY", testKey)

		notADirectory := filepath.Join(t.TempDir(), "file")
		if err := os.WriteFile(notADirectory, nil, 0o600); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}

		storage := newFileStorage(filepath.Join(notADirectory, secretsFilename), []byte(testKey))
		if _, err := storage.List(t.Context()); err == nil {
			t.Error("List: expected an error for a workspace which isn't a directory")
		}
	})

	// The storage API takes a context so a driver talking to a remote host
	// can be cancelled; the file driver honours it too.
	t.Run("cancelled context", func(t *testing.T) {
		storage := newTestStorage(t)
		seed(t, storage, "DB_DSN", "user:password@hostname")

		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		if _, err := storage.List(ctx); !errors.Is(err, context.Canceled) {
			t.Errorf("List: %v, want %v", err, context.Canceled)
		}
		if _, err := storage.Get(ctx, "DB_DSN"); !errors.Is(err, context.Canceled) {
			t.Errorf("Get: %v, want %v", err, context.Canceled)
		}
		if _, err := storage.Set(ctx, "DB_DSN", "newvalue"); !errors.Is(err, context.Canceled) {
			t.Errorf("Set: %v, want %v", err, context.Canceled)
		}

		// The read is what stops a cancelled Set above. This is the other
		// half: the cli cancels on SIGINT, which can land after the file was
		// read and before it's written.
		if err := storage.save(ctx); !errors.Is(err, context.Canceled) {
			t.Errorf("save: %v, want %v", err, context.Canceled)
		}
	})
}
