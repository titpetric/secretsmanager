package logger

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

// TestNew covers the lines the cli reports through.
func TestNew(t *testing.T) {
	var out bytes.Buffer

	log := New(&out)
	log.Info("build", "version", "139d40bb", "time", "2026-08-20T12:46:53Z")
	log.Error("exiting", "error", "no such secret")

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("logged %d lines, want 2:\n%s", len(lines), out.String())
	}

	for i, want := range []string{
		`level=INFO msg=build version=139d40bb time=2026-08-20T12:46:53Z`,
		`level=ERROR msg=exiting error="no such secret"`,
	} {
		if lines[i] != want {
			t.Errorf("line %d = %q, want %q", i, lines[i], want)
		}
	}
}

// TestNewWithoutTimestamp covers the timestamp a handler writes by default,
// which a command running in front of someone doesn't need.
func TestNewWithoutTimestamp(t *testing.T) {
	var out bytes.Buffer

	New(&out).Info("stored")

	if got, want := strings.TrimSpace(out.String()), "level=INFO msg=stored"; got != want {
		t.Errorf("logged %q, want %q", got, want)
	}
}

// TestNewWithAttrs covers a logger carrying attributes, which has to keep
// dropping the timestamp: the handler underneath returns itself from With,
// and that would leave the wrapping behind.
func TestNewWithAttrs(t *testing.T) {
	var out bytes.Buffer

	log := New(&out).With("command", "get")
	log.Info("stored")
	log.WithGroup("secret").Info("read", "name", "DB_DSN")

	want := "level=INFO msg=stored command=get\n" +
		"level=INFO msg=read command=get secret.name=DB_DSN\n"
	if got := out.String(); got != want {
		t.Errorf("logged:\n%s\nwant:\n%s", got, want)
	}
}

// TestNewKeepsATimeAttribute covers an attribute which shares the name the
// handler writes its timestamp under. Dropping it by name alone would eat
// the build time the version command reports.
func TestNewKeepsATimeAttribute(t *testing.T) {
	var out bytes.Buffer

	when := time.Date(2026, time.August, 20, 12, 46, 53, 0, time.UTC)
	New(&out).Info("build", "time", when)

	if !strings.Contains(out.String(), "time=2026-08-20T12:46:53") {
		t.Errorf("logged %q, want it to keep the time it was given", out.String())
	}
}
