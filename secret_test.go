package secretsmanager

import (
	"os"
	"strings"
	"testing"

	"github.com/titpetric/secretsmanager/internal/ulid"
)

// TestSecret covers one secret, which the storage encrypts and decrypts
// around.
func TestSecret(t *testing.T) {
	t.Run("new secrets have a ULID", func(t *testing.T) {
		secret := newSecret("DB_DSN", "user:password@hostname")

		if !ulid.Valid(secret.ID) {
			t.Errorf("ID %q is not a ULID", secret.ID)
		}
		if got, want := secret.Name, "DB_DSN"; got != want {
			t.Errorf("Name = %q, want %q", got, want)
		}
		if got, want := secret.Value, "user:password@hostname"; got != want {
			t.Errorf("Value = %q, want %q", got, want)
		}
		if secret.raw != "" {
			t.Errorf("raw = %q, want it empty until the secret is written", secret.raw)
		}
	})

	// Files written before ULIDs hold a UUID in the same field. They have to
	// keep loading, and keep their ID.
	t.Run("a legacy UUID is kept", func(t *testing.T) {
		const legacyID = "25349927-99b2-4ac5-ad59-d63f88f4a612"

		storage := newTestStorage(t)
		contents := seed(t, storage, "DB_DSN", "user:password@hostname")

		contents = []byte(strings.Replace(string(contents), list(t, storage)[0].ID, legacyID, 1))
		if err := os.WriteFile(storage.filename, contents, 0o600); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}

		writer := newFileStorage(storage.filename, []byte(testKey))
		if got := list(t, writer)[0].ID; got != legacyID {
			t.Errorf("ID = %q, want %q", got, legacyID)
		}
		if _, err := writer.Set(t.Context(), "API_KEY", "abc123"); err != nil {
			t.Fatalf("Set: %v", err)
		}

		reader := newFileStorage(storage.filename, []byte(testKey))
		if got := list(t, reader)[0].ID; got != legacyID {
			t.Errorf("ID after a write = %q, want %q", got, legacyID)
		}
	})
}
