package secretsmanager

import (
	"strings"
	"testing"
)

// TestEncrypt covers the encryption the values are stored under.
func TestEncrypt(t *testing.T) {
	key := []byte(testKey)

	t.Run("round trip", func(t *testing.T) {
		for _, value := range []string{
			"",
			"user:password@hostname",
			"  padded  ",
			"multi\nline",
			strings.Repeat("long", 1000),
			"\x00\xff binary",
		} {
			encrypted, err := encrypt(key, value)
			if err != nil {
				t.Fatalf("encrypt(%q): %v", value, err)
			}
			if strings.Contains(encrypted, value) && value != "" {
				t.Errorf("encrypt(%q) = %q, which holds the value in the clear", value, encrypted)
			}

			decrypted, err := decrypt(key, encrypted)
			if err != nil {
				t.Fatalf("decrypt: %v", err)
			}
			if decrypted != value {
				t.Errorf("round trip of %q gave %q", value, decrypted)
			}
		}
	})

	// The IV is new every time, which is why the storage keeps the
	// ciphertext of a value it hasn't changed.
	t.Run("the same value encrypts differently", func(t *testing.T) {
		first, err := encrypt(key, "user:password@hostname")
		if err != nil {
			t.Fatalf("encrypt: %v", err)
		}
		second, err := encrypt(key, "user:password@hostname")
		if err != nil {
			t.Fatalf("encrypt: %v", err)
		}
		if first == second {
			t.Error("encrypt gave the same ciphertext twice, so the IV isn't new")
		}
	})

	t.Run("keys of the wrong length", func(t *testing.T) {
		if _, err := encrypt([]byte("short"), "value"); err == nil {
			t.Error("encrypt: expected an error for a key AES can't take")
		}
		if _, err := decrypt([]byte("short"), "AAAAAAAAAAAAAAAAAAAAAAA="); err == nil {
			t.Error("decrypt: expected an error for a key AES can't take")
		}
	})

	t.Run("damaged values", func(t *testing.T) {
		if _, err := decrypt(key, "not base64!"); err == nil {
			t.Error("decrypt: expected an error for a value which isn't base64")
		}
		if _, err := decrypt(key, "AAAA"); err == nil {
			t.Error("decrypt: expected an error for a value too short to hold an IV")
		}
	})

	// Nothing authenticates the value, so the wrong key reads as rubbish
	// rather than as an error. Worth pinning: it's why a mistyped key gives
	// junk secrets instead of a refusal.
	t.Run("the wrong key gives the wrong value", func(t *testing.T) {
		encrypted, err := encrypt(key, "user:password@hostname")
		if err != nil {
			t.Fatalf("encrypt: %v", err)
		}

		decrypted, err := decrypt([]byte(strings.ToUpper(testKey)), encrypted)
		if err != nil {
			t.Fatalf("decrypt: %v", err)
		}
		if decrypted == "user:password@hostname" {
			t.Error("a value encrypted with another key came back intact")
		}
	})
}

// TestDecryptStoredFormat reads a value written by the version which used
// go-cryptkeeper, so the files that version wrote keep working.
func TestDecryptStoredFormat(t *testing.T) {
	const (
		stored = "0mT85-AsvIYeyfNkjp_PE3dNoqIHCzO1NQHg-Y3iECoY4Y6DLY4="
		want   = "user:password@hostname"
	)

	got, err := decrypt([]byte(testKey), stored)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if got != want {
		t.Errorf("decrypt(%q) = %q, want %q", stored, got, want)
	}
}
