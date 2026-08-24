package main

import (
	"io"
	"strings"
	"testing"

	"github.com/titpetric/secretsmanager/internal/logger"
)

// TestStart covers the wiring, which fails before any command runs when the
// workspace names a driver that doesn't exist.
func TestStart(t *testing.T) {
	t.Setenv("SECRETSMANAGER_WORKSPACE", "https://secrets.example.com/team")

	err := start(logger.New(io.Discard))
	if err == nil {
		t.Fatal("start: expected an error for a workspace which isn't a directory")
	}
	if !strings.Contains(err.Error(), "isn't implemented") {
		t.Errorf("start: %v, want the error from the driver lookup", err)
	}
}
