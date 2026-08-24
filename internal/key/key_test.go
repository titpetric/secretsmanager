package key

import (
	"strings"
	"testing"
)

// testKey is a key of the length FromEnv insists on.
const testKey = "0123456789abcdef0123456789abcdef"

// TestFromEnv covers the key a storage falls back to.
func TestFromEnv(t *testing.T) {
	t.Setenv("SECRETSMANAGER_KEY", "")
	if _, err := FromEnv(); err == nil {
		t.Error("FromEnv: expected an error for a key which isn't set")
	}

	t.Setenv("SECRETSMANAGER_KEY", "tooshort")
	if _, err := FromEnv(); err == nil {
		t.Error("FromEnv: expected an error for a key of the wrong length")
	}

	t.Setenv("SECRETSMANAGER_KEY", testKey)
	got, err := FromEnv()
	if err != nil {
		t.Fatalf("FromEnv: %v", err)
	}
	if string(got) != testKey {
		t.Errorf("FromEnv() = %q, want %q", got, testKey)
	}
}

// TestGenerate covers the key init prints.
func TestGenerate(t *testing.T) {
	seen := make(map[string]bool, 100)

	for range 100 {
		generated := Generate()

		if len(generated) != Length {
			t.Fatalf("key %q is %d characters, want %d", generated, len(generated), Length)
		}
		if seen[generated] {
			t.Fatalf("key %q came up twice", generated)
		}
		seen[generated] = true

		// Nothing which is read back as another character, and nothing a
		// shell would act on: the key is printed to be copied by hand.
		if i := strings.IndexFunc(generated, func(r rune) bool {
			return !strings.ContainsRune(alphabet, r)
		}); i >= 0 {
			t.Errorf("key %q holds %q, which isn't in the alphabet", generated, generated[i])
		}
	}

	// And it works as a key.
	t.Setenv("SECRETSMANAGER_KEY", Generate())
	if _, err := FromEnv(); err != nil {
		t.Errorf("FromEnv: %v", err)
	}
}
