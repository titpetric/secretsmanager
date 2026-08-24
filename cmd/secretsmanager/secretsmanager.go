package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/titpetric/cli"
	"github.com/titpetric/secretsmanager"
	"github.com/titpetric/secretsmanager/internal/envname"
	"github.com/titpetric/secretsmanager/internal/key"
)

// Logger is where a command reports what it did, as opposed to what it
// prints for a script to read. It's the part of *slog.Logger the commands
// use, named as an interface so a caller can report through its own.
type Logger interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
}

// SecretsManager binds the cli commands to a storage driver. One is built
// per run, and owns what it reads the prompts from and what it reports on.
type SecretsManager struct {
	secretsmanager.Storage

	// in is where create reads a name and a value from. It's buffered, and
	// the buffer is why it's held here rather than made per prompt: a new
	// reader drops whatever it read past the newline.
	in *bufio.Reader

	log Logger
}

// NewSecretsManager returns a manager for the configured workspace.
func NewSecretsManager(log Logger) (*SecretsManager, error) {
	storage, err := secretsmanager.NewStorage(secretsmanager.NewOptionsFromEnv())
	if err != nil {
		return nil, err
	}

	return &SecretsManager{
		Storage: storage,
		in:      bufio.NewReader(os.Stdin),
		log:     log,
	}, nil
}

// Commands returns the cli commands, in the order help lists them.
func (s *SecretsManager) Commands() []*cli.CommandInfo {
	return []*cli.CommandInfo{
		s.init(),
		s.createSecret(),
		s.getSecret(),
		s.environment(),
		s.version(),
	}
}

// createSecret returns the create command, which stores one secret, asking
// for its name and value.
func (s *SecretsManager) createSecret() *cli.CommandInfo {
	return &cli.CommandInfo{
		Name:  "create",
		Title: "Create Secret",
		New: func() *cli.Command {
			return &cli.Command{
				Run: func(ctx context.Context, _ []string) error {
					name, err := s.readName("Name for your new secret:")
					if err != nil {
						return err
					}
					if !envname.Valid(envname.Name(name)) {
						return fmt.Errorf("%q doesn't produce a usable environment variable name (%q)", name, envname.Name(name))
					}

					status := "Created new secret"
					switch _, err := s.Get(ctx, name); {
					case err == nil:
						status = "Updated existing secret"
					case !errors.Is(err, secretsmanager.ErrNotFound):
						return err
					}

					value, err := s.readLine("Secret value:")
					if err != nil {
						return err
					}

					secret, err := s.Set(ctx, name, value)
					if err != nil {
						return err
					}

					fmt.Printf("%s:\n\nID: %s\nName: %s\nValue: %s\n", status, secret.ID, secret.Name, secret.Value)
					return nil
				},
			}
		},
	}
}

// getSecret returns the get command, which prints the value of one secret
// for a script to read.
func (s *SecretsManager) getSecret() *cli.CommandInfo {
	return &cli.CommandInfo{
		Name:  "get",
		Title: "Print the value of a secret",
		New: func() *cli.Command {
			return &cli.Command{
				Usage: func() string {
					return "Usage: secretsmanager get NAME"
				},
				Run: func(ctx context.Context, args []string) error {
					if len(args) != 1 {
						return errors.New("expected the name of one secret")
					}

					secret, err := s.Get(ctx, args[0])
					if err != nil {
						return err
					}

					fmt.Println(secret.Value)
					return nil
				},
			}
		},
	}
}

// environment returns the env command, which prints every secret in the
// form a .env file takes.
func (s *SecretsManager) environment() *cli.CommandInfo {
	return &cli.CommandInfo{
		Name:  "env",
		Title: "Environment Secrets",
		New: func() *cli.Command {
			return &cli.Command{
				Run: func(ctx context.Context, _ []string) error {
					secrets, err := s.List(ctx)
					if err != nil {
						return err
					}

					for _, secret := range secrets {
						name := envname.Name(secret.Name)
						if !envname.Valid(name) {
							return fmt.Errorf("secret %q is stored as %q, which isn't a usable environment variable name", secret.Name, name)
						}

						fmt.Printf("%s=%s\n", name, shellQuote(secret.Value))
					}
					return nil
				},
			}
		},
	}
}

// init returns the init command, which generates an encryption key and
// names the workspace to read the secrets from.
func (s *SecretsManager) init() *cli.CommandInfo {
	return &cli.CommandInfo{
		Name:  "init",
		Title: "Generate encryption key",
		New: func() *cli.Command {
			return &cli.Command{
				Run: func(_ context.Context, _ []string) error {
					if os.Getenv("SECRETSMANAGER_KEY") != "" {
						return errors.New("SECRETSMANAGER_KEY already exists")
					}

					// Both variables are the configuration: the key to read
					// the secrets with, and where to read them from. Without
					// the workspace the tool only works from one directory.
					fmt.Println("# Add the following to /etc/environment and store securely in case you need to restore")
					fmt.Println("# WARN: Please, don't add/commit this key to git, as it allows decrypting all secrets.")
					fmt.Println("SECRETSMANAGER_KEY=" + key.Generate())
					fmt.Println("# The directory holding .secrets.json, so the tool works from anywhere.")
					fmt.Println("SECRETSMANAGER_WORKSPACE=" + secretsmanager.NewOptionsFromEnv().Workspace)
					return nil
				},
			}
		},
	}
}
