package secretsmanager

import (
	"os"
	"strings"
	"testing"
)

// testKey is a valid encryption key, of the length Load insists on.
const testKey = "0123456789abcdef0123456789abcdef"

// TestNewStorage covers the driver a workspace selects.
func TestNewStorage(t *testing.T) {
	for workspace, want := range map[string]string{
		"":           secretsFilename,
		"/workspace": "/workspace/.secrets.json",
		"../shared":  "../shared/.secrets.json",
		// A directory which happens to hold :// is still a directory. The
		// doubled separator is cleaned away, which names the same file.
		"/tmp/od://d": "/tmp/od:/d/.secrets.json",
	} {
		storage, err := NewStorage(Options{Workspace: workspace})
		if err != nil {
			t.Errorf("NewStorage(%q): %v", workspace, err)
			continue
		}
		if got := storage.String(); got != want {
			t.Errorf("NewStorage(%q) = %q, want %q", workspace, got, want)
		}
	}

	// A remote driver would be selected here, once there is one.
	storage, err := NewStorage(Options{Workspace: "https://secrets.example.com/team"})
	if err == nil {
		t.Fatalf("NewStorage = %v, want an error for a workspace which isn't a directory", storage)
	}
	if !strings.Contains(err.Error(), "https") {
		t.Errorf("NewStorage: %v, want the error to name the scheme", err)
	}
}

// TestOptions covers what the zero value configures, which is what the doc
// comment promises: the current directory, with the key from the
// environment.
func TestOptions(t *testing.T) {
	t.Setenv("SECRETSMANAGER_KEY", testKey)
	t.Chdir(t.TempDir())

	storage, err := NewStorage(Options{})
	if err != nil {
		t.Fatalf("NewStorage: %v", err)
	}
	if got, want := storage.String(), secretsFilename; got != want {
		t.Errorf("storage = %q, want %q, which is the current directory", got, want)
	}

	if _, err := storage.Set(t.Context(), "DB_DSN", "user:password@hostname"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if _, err := os.Stat(secretsFilename); err != nil {
		t.Errorf("Stat: %v, want the file written in the current directory", err)
	}

	secret, err := storage.Get(t.Context(), "DB_DSN")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got, want := secret.Value, "user:password@hostname"; got != want {
		t.Errorf("Value = %q, want %q", got, want)
	}
}

// TestNewOptionsFromEnv covers the configuration the environment carries.
func TestNewOptionsFromEnv(t *testing.T) {
	t.Setenv("SECRETSMANAGER_WORKSPACE", "/workspace")
	t.Setenv("SECRETSMANAGER_KEY", testKey)

	options := NewOptionsFromEnv()
	if got, want := options.Workspace, "/workspace"; got != want {
		t.Errorf("Workspace = %q, want %q", got, want)
	}
	if got := string(options.Key); got != testKey {
		t.Errorf("Key = %q, want %q", got, testKey)
	}

	// A key which isn't usable is left for the storage to report, so init
	// can run before there is one.
	for _, unusable := range []string{"", "tooshort"} {
		t.Setenv("SECRETSMANAGER_KEY", unusable)
		if key := NewOptionsFromEnv().Key; key != nil {
			t.Errorf("Key = %q for %q, want it left unset", key, unusable)
		}
	}
}

// TestWorkspace covers reading the workspace from the environment, which
// falls back to the directory the caller is running in.
func TestWorkspace(t *testing.T) {
	t.Setenv("SECRETSMANAGER_WORKSPACE", " /workspace ")
	if got, want := workspace(), "/workspace"; got != want {
		t.Errorf("workspace() = %q, want %q", got, want)
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}

	for _, unset := range []string{"", "   "} {
		t.Setenv("SECRETSMANAGER_WORKSPACE", unset)
		if got := workspace(); got != wd {
			t.Errorf("workspace() = %q, want the working directory %q", got, wd)
		}
	}
}
