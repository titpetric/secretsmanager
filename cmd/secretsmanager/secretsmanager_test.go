package main

import (
	"bufio"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/titpetric/cli"
	"github.com/titpetric/secretsmanager"
	"github.com/titpetric/secretsmanager/internal/logger"
)

// testKey is a valid encryption key, of the length the storage insists on.
const testKey = "0123456789abcdef0123456789abcdef"

// newTestManager returns a manager over a workspace in a temporary
// directory, with the encryption key set for the duration of the test.
func newTestManager(t *testing.T) *SecretsManager {
	t.Helper()
	t.Setenv("SECRETSMANAGER_KEY", testKey)

	storage, err := secretsmanager.NewStorage(secretsmanager.Options{Workspace: t.TempDir()})
	if err != nil {
		t.Fatalf("NewStorage: %v", err)
	}
	return &SecretsManager{
		Storage: storage,
		in:      bufio.NewReader(strings.NewReader("")),
		log:     logger.New(io.Discard),
	}
}

// newEnvManager returns a manager built the way the binary builds it, from
// the environment, for the commands which report the configuration back.
func newEnvManager(t *testing.T) *SecretsManager {
	t.Helper()
	silenceStderr(t)

	manager, err := NewSecretsManager(logger.New(io.Discard))
	if err != nil {
		t.Fatalf("NewSecretsManager: %v", err)
	}
	return manager
}

// setStdin gives the manager its input, and drops the prompts it writes to
// stderr for the duration of the test.
func setStdin(t *testing.T, manager *SecretsManager, input string) {
	t.Helper()

	manager.in = bufio.NewReader(strings.NewReader(input))
	silenceStderr(t)
}

// silenceStderr drops what the test writes to stderr, which is the prompts
// and the log lines.
func silenceStderr(t *testing.T) {
	t.Helper()

	discard, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatalf("OpenFile: %v", err)
	}
	saved := os.Stderr
	os.Stderr = discard

	t.Cleanup(func() {
		os.Stderr = saved
		discard.Close()
	})
}

// run executes a command by name and returns what it printed. Prompts and
// log lines are written to stderr, which is dropped.
func run(t *testing.T, manager *SecretsManager, name string, args ...string) (string, error) {
	t.Helper()

	var command *cli.Command
	for _, info := range manager.Commands() {
		if info.Name == name {
			command = info.New()
		}
	}
	if command == nil {
		t.Fatalf("no command named %q", name)
	}

	stdout, stderr := os.Stdout, os.Stderr
	out, err := os.CreateTemp(t.TempDir(), "stdout")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	discard, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatalf("OpenFile: %v", err)
	}

	os.Stdout, os.Stderr = out, discard
	runErr := command.Run(t.Context(), args)
	os.Stdout, os.Stderr = stdout, stderr
	discard.Close()

	if _, err := out.Seek(0, io.SeekStart); err != nil {
		t.Fatalf("Seek: %v", err)
	}
	printed, err := io.ReadAll(out)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	out.Close()

	return string(printed), runErr
}

// setting returns the value printed for an environment variable, from
// output made of NAME=value lines and comments.
func setting(t *testing.T, out, name string) string {
	t.Helper()

	for line := range strings.SplitSeq(out, "\n") {
		if value, ok := strings.CutPrefix(line, name+"="); ok {
			return value
		}
	}

	t.Fatalf("output doesn't set %s:\n%s", name, out)
	return ""
}

// TestSecretsManager covers the commands, over a storage in a temporary
// directory.
func TestSecretsManager(t *testing.T) {
	t.Run("storage comes from the workspace", func(t *testing.T) {
		workspace := t.TempDir()
		t.Setenv("SECRETSMANAGER_WORKSPACE", workspace)

		manager, err := NewSecretsManager(logger.New(io.Discard))
		if err != nil {
			t.Fatalf("NewSecretsManager: %v", err)
		}
		if got, want := manager.Storage.String(), filepath.Join(workspace, ".secrets.json"); got != want {
			t.Errorf("storage = %q, want %q", got, want)
		}
		if len(manager.Commands()) == 0 {
			t.Error("Commands: no commands registered")
		}

		t.Setenv("SECRETSMANAGER_WORKSPACE", "https://secrets.example.com/team")
		if _, err := NewSecretsManager(logger.New(io.Discard)); err == nil {
			t.Error("NewSecretsManager: expected an error for a workspace which isn't a directory")
		}
	})

	t.Run("create stores a secret", func(t *testing.T) {
		manager := newTestManager(t)

		setStdin(t, manager, "DB_DSN\nuser:password@hostname\n")
		out, err := run(t, manager, "create")
		if err != nil {
			t.Fatalf("create: %v", err)
		}
		if !strings.Contains(out, "Created new secret") {
			t.Errorf("create printed %q, want it to report a new secret", out)
		}

		secret, err := manager.Get(t.Context(), "DB_DSN")
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		if got, want := secret.Value, "user:password@hostname"; got != want {
			t.Errorf("Value = %q, want %q", got, want)
		}
	})

	t.Run("create reports an update", func(t *testing.T) {
		manager := newTestManager(t)

		setStdin(t, manager, "DB_DSN\nuser:password@hostname\n")
		if _, err := run(t, manager, "create"); err != nil {
			t.Fatalf("create: %v", err)
		}

		setStdin(t, manager, "db-dsn\nnewvalue\n")
		out, err := run(t, manager, "create")
		if err != nil {
			t.Fatalf("create: %v", err)
		}
		if !strings.Contains(out, "Updated existing secret") {
			t.Errorf("create printed %q, want it to report an update", out)
		}
	})

	t.Run("create refuses an unusable name", func(t *testing.T) {
		manager := newTestManager(t)

		setStdin(t, manager, "1foo\nvalue\n")
		if _, err := run(t, manager, "create"); err == nil {
			t.Error("create: expected an error for a name which isn't an environment name")
		}
	})

	t.Run("create stops at the end of the input", func(t *testing.T) {
		manager := newTestManager(t)

		setStdin(t, manager, "DB_DSN\n")
		if _, err := run(t, manager, "create"); err == nil {
			t.Error("create: expected an error when no value was given")
		}

		setStdin(t, manager, "")
		if _, err := run(t, manager, "create"); err == nil {
			t.Error("create: expected an error when no name was given")
		}
	})

	// The name is read before the file, so a file which can't be read fails
	// the lookup rather than the prompt.
	t.Run("create reports a file it can't read", func(t *testing.T) {
		manager := newTestManager(t)

		if err := os.WriteFile(manager.Storage.String(), []byte("{"), 0o600); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}

		setStdin(t, manager, "DB_DSN\nuser:password@hostname\n")
		if _, err := run(t, manager, "create"); err == nil {
			t.Error("create: expected an error for a file which isn't JSON")
		}
	})

	t.Run("get prints a value", func(t *testing.T) {
		manager := newTestManager(t)

		setStdin(t, manager, "DB_DSN\nuser:password@hostname\n")
		if _, err := run(t, manager, "create"); err != nil {
			t.Fatalf("create: %v", err)
		}

		out, err := run(t, manager, "get", "db-dsn")
		if err != nil {
			t.Fatalf("get: %v", err)
		}
		if got, want := out, "user:password@hostname\n"; got != want {
			t.Errorf("get printed %q, want %q", got, want)
		}
	})

	t.Run("get needs one name", func(t *testing.T) {
		manager := newTestManager(t)

		if _, err := run(t, manager, "get"); err == nil {
			t.Error("get: expected an error without a name")
		}
		if _, err := run(t, manager, "get", "ONE", "TWO"); err == nil {
			t.Error("get: expected an error for more than one name")
		}

		// Which is what the usage line printed with the help says to do.
		usage := manager.getSecret().New().Usage()
		if !strings.Contains(usage, "get NAME") {
			t.Errorf("usage = %q, want it to show one name", usage)
		}
	})

	t.Run("get reports a missing secret", func(t *testing.T) {
		manager := newTestManager(t)

		_, err := run(t, manager, "get", "MISSING")
		if !errors.Is(err, secretsmanager.ErrNotFound) {
			t.Errorf("get: %v, want %v", err, secretsmanager.ErrNotFound)
		}
	})

	t.Run("env prints every secret", func(t *testing.T) {
		manager := newTestManager(t)

		setStdin(t, manager, "DB_DSN\nuser:pass`word$\n")
		if _, err := run(t, manager, "create"); err != nil {
			t.Fatalf("create: %v", err)
		}
		setStdin(t, manager, "API_KEY\nabc123\n")
		if _, err := run(t, manager, "create"); err != nil {
			t.Fatalf("create: %v", err)
		}

		out, err := run(t, manager, "env")
		if err != nil {
			t.Fatalf("env: %v", err)
		}

		want := "DB_DSN=\"user:pass\\`word\\$\"\nAPI_KEY=\"abc123\"\n"
		if out != want {
			t.Errorf("env printed %q, want %q", out, want)
		}
	})

	// The names are safe to print where the values aren't, so list is what
	// to run to see what's stored without spilling any of it.
	t.Run("list prints the names without the values", func(t *testing.T) {
		manager := newTestManager(t)

		setStdin(t, manager, "DB_DSN\nuser:password\n")
		if _, err := run(t, manager, "create"); err != nil {
			t.Fatalf("create: %v", err)
		}
		setStdin(t, manager, "API_KEY\nabc123\n")
		if _, err := run(t, manager, "create"); err != nil {
			t.Fatalf("create: %v", err)
		}

		out, err := run(t, manager, "list")
		if err != nil {
			t.Fatalf("list: %v", err)
		}

		want := "DB_DSN\nAPI_KEY\n"
		if out != want {
			t.Errorf("list printed %q, want %q", out, want)
		}
	})

	t.Run("list without a key", func(t *testing.T) {
		manager := newTestManager(t)
		t.Setenv("SECRETSMANAGER_KEY", "")

		if _, err := run(t, manager, "list"); err == nil {
			t.Error("list: expected an error for a missing SECRETSMANAGER_KEY")
		}
	})

	// A value may be padded with spaces on purpose. Trimming it would store
	// a secret nobody asked for, and the padding survives a .env round trip
	// because the value is quoted.
	t.Run("create keeps the whitespace in a value", func(t *testing.T) {
		manager := newTestManager(t)

		setStdin(t, manager, "DB_DSN\n  padded  \n")
		if _, err := run(t, manager, "create"); err != nil {
			t.Fatalf("create: %v", err)
		}

		out, err := run(t, manager, "get", "DB_DSN")
		if err != nil {
			t.Fatalf("get: %v", err)
		}
		if got, want := out, "  padded  \n"; got != want {
			t.Errorf("get printed %q, want %q", got, want)
		}
	})

	// A name can be edited into the file by hand, and env has to print a
	// usable environment variable name or refuse to print at all.
	t.Run("env refuses an unusable name", func(t *testing.T) {
		manager := newTestManager(t)

		if _, err := manager.Set(t.Context(), "foo!bar", "value"); err != nil {
			t.Fatalf("Set: %v", err)
		}

		out, err := run(t, manager, "env")
		if err == nil {
			t.Fatalf("env printed %q, want an error for a name which isn't an environment name", out)
		}
		if !strings.Contains(err.Error(), "foo!bar") {
			t.Errorf("env: %v, want the error to name the secret", err)
		}
	})

	t.Run("env without a key", func(t *testing.T) {
		manager := newTestManager(t)
		t.Setenv("SECRETSMANAGER_KEY", "")

		if _, err := run(t, manager, "env"); err == nil {
			t.Error("env: expected an error for a missing SECRETSMANAGER_KEY")
		}
	})

	// The output is the whole configuration: what to decrypt with, and
	// where to read the secrets from.
	t.Run("init generates a key and names the workspace", func(t *testing.T) {
		workspace := t.TempDir()
		t.Setenv("SECRETSMANAGER_KEY", "")
		t.Setenv("SECRETSMANAGER_WORKSPACE", workspace)

		manager := newEnvManager(t)

		out, err := run(t, manager, "init")
		if err != nil {
			t.Fatalf("init: %v", err)
		}

		if got := len(setting(t, out, "SECRETSMANAGER_KEY")); got != 32 {
			t.Errorf("init generated a key of %d characters, want 32", got)
		}
		if got := setting(t, out, "SECRETSMANAGER_WORKSPACE"); got != workspace {
			t.Errorf("init reported the workspace as %q, want %q", got, workspace)
		}
	})

	// Without one configured, the workspace to report is the directory the
	// command ran in, which is where create would write the file.
	t.Run("init falls back to the current directory", func(t *testing.T) {
		t.Setenv("SECRETSMANAGER_KEY", "")
		t.Setenv("SECRETSMANAGER_WORKSPACE", "")

		manager := newEnvManager(t)

		out, err := run(t, manager, "init")
		if err != nil {
			t.Fatalf("init: %v", err)
		}

		wd, err := os.Getwd()
		if err != nil {
			t.Fatalf("Getwd: %v", err)
		}
		if got := setting(t, out, "SECRETSMANAGER_WORKSPACE"); got != wd {
			t.Errorf("init reported the workspace as %q, want %q", got, wd)
		}
	})

	// Generating a key over an existing one is allowed, but the output
	// has to say that the secrets encrypted with the old key are lost.
	t.Run("init warns about an existing key", func(t *testing.T) {
		manager := newTestManager(t)

		out, err := run(t, manager, "init")
		if err != nil {
			t.Fatalf("init: %v", err)
		}

		if !strings.Contains(out, "SECRETSMANAGER_KEY is already set") {
			t.Errorf("init printed %q, want a warning about the existing key", out)
		}
		if got := len(setting(t, out, "SECRETSMANAGER_KEY")); got != 32 {
			t.Errorf("init generated a key of %d characters, want 32", got)
		}
	})

	t.Run("version", func(t *testing.T) {
		manager := newTestManager(t)

		if _, err := run(t, manager, "version"); err != nil {
			t.Errorf("version: %v", err)
		}
	})
}
