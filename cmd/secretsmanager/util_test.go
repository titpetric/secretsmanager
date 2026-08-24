package main

import (
	"bufio"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/titpetric/secretsmanager/internal/logger"
)

// errReader is what failingReader fails with.
var errReader = errors.New("read failed")

// failingReader stands in for input which can't be read at all, which is
// neither a value nor the end of the input.
type failingReader struct{}

// Read always fails.
func (failingReader) Read([]byte) (int, error) {
	return 0, errReader
}

// readerFor returns a manager reading from input, with the prompts it writes
// to stderr dropped. It needs no storage: reading a line doesn't touch one.
func readerFor(t *testing.T, input io.Reader) *SecretsManager {
	t.Helper()
	silenceStderr(t)

	return &SecretsManager{in: bufio.NewReader(input), log: logger.New(io.Discard)}
}

// TestReadLine covers reading a value from the input, which is piped in as
// often as it is typed.
func TestReadLine(t *testing.T) {
	t.Run("reads a value per call", func(t *testing.T) {
		manager := readerFor(t, strings.NewReader("DB_DSN\nuser:password@hostname\n"))

		for _, want := range []string{"DB_DSN", "user:password@hostname"} {
			got, err := manager.readLine("Value:")
			if err != nil {
				t.Fatalf("readLine: %v", err)
			}
			if got != want {
				t.Errorf("readLine = %q, want %q", got, want)
			}
		}
	})

	t.Run("skips blank lines", func(t *testing.T) {
		manager := readerFor(t, strings.NewReader("\n   \nDB_DSN\n"))

		got, err := manager.readLine("Name:")
		if err != nil {
			t.Fatalf("readLine: %v", err)
		}
		if want := "DB_DSN"; got != want {
			t.Errorf("readLine = %q, want %q", got, want)
		}
	})

	// A secret may be padded with spaces on purpose, so only the line ending
	// is removed. The name is read with readName, which does trim.
	t.Run("keeps the whitespace around a value", func(t *testing.T) {
		manager := readerFor(t, strings.NewReader("  padded value  \r\n"))

		got, err := manager.readLine("Value:")
		if err != nil {
			t.Fatalf("readLine: %v", err)
		}
		if want := "  padded value  "; got != want {
			t.Errorf("readLine = %q, want %q", got, want)
		}
	})

	t.Run("the end of the input is an error", func(t *testing.T) {
		manager := readerFor(t, strings.NewReader(""))

		if _, err := manager.readLine("Name:"); err == nil {
			t.Error("readLine: expected an error at the end of the input")
		}
	})

	t.Run("a failing reader is reported", func(t *testing.T) {
		manager := readerFor(t, failingReader{})

		_, err := manager.readLine("Name:")
		if !errors.Is(err, errReader) {
			t.Errorf("readLine: %v, want %v", err, errReader)
		}
	})

	// A value on the last line without a newline after it is still a value.
	t.Run("a value without a newline", func(t *testing.T) {
		manager := readerFor(t, strings.NewReader("DB_DSN"))

		got, err := manager.readLine("Name:")
		if err != nil {
			t.Fatalf("readLine: %v", err)
		}
		if want := "DB_DSN"; got != want {
			t.Errorf("readLine = %q, want %q", got, want)
		}
	})

	// Each manager reads its own input, so one taking over the terminal
	// doesn't leave the next one with what it buffered.
	t.Run("two managers read their own input", func(t *testing.T) {
		first := readerFor(t, strings.NewReader("FIRST\n"))
		second := readerFor(t, strings.NewReader("SECOND\n"))

		got, err := second.readLine("Name:")
		if err != nil {
			t.Fatalf("readLine: %v", err)
		}
		if want := "SECOND"; got != want {
			t.Errorf("readLine = %q, want %q", got, want)
		}
		if got, _ := first.readLine("Name:"); got != "FIRST" {
			t.Errorf("readLine = %q, want %q", got, "FIRST")
		}
	})
}

// TestReadName covers reading a name, which is trimmed.
func TestReadName(t *testing.T) {
	manager := readerFor(t, strings.NewReader("  DB_DSN  \n"))

	got, err := manager.readName("Name:")
	if err != nil {
		t.Fatalf("readName: %v", err)
	}
	if want := "DB_DSN"; got != want {
		t.Errorf("readName = %q, want %q", got, want)
	}
}
