package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/go-bridget/mig/cli"
	"github.com/iancoleman/strcase"
	"github.com/m1/go-generate-password/generator"
)

type SecretsManager struct {
	*Storage
}

func NewSecretsManager() (*SecretsManager, error) {
	storage, err := NewStorage(".secrets.json")
	if err != nil {
		return nil, err
	}

	return &SecretsManager{
		Storage: storage,
	}, nil
}

func (s *SecretsManager) Commands() []*cli.CommandInfo {
	return []*cli.CommandInfo{
		s.init(),
		s.createSecret(),
		s.environment(),
		version,
	}
}

func (s *SecretsManager) createSecret() *cli.CommandInfo {
	return &cli.CommandInfo{
		Name:  "create",
		Title: "Create Secret",
		New: func() *cli.Command {
			return &cli.Command{
				Run: func(_ context.Context, _ []string) error {
					if err := s.Load(); err != nil {
						return err
					}

					name := readline("Name for your new secret:")
					value := readline("Secret value:")

					secret, err := NewSecret(name, value)
					if err != nil {
						return err
					}

					fmt.Printf("Created new secret:\n\nID: %s\nName: %s\nValue: %s\n", secret.ID, secret.Name, secret.Value.String)

					s.Storage.Secrets = append(s.Storage.Secrets, secret)
					return nil
				},
			}
		},
	}
}

func (s *SecretsManager) environment() *cli.CommandInfo {
	return &cli.CommandInfo{
		Name:  "env",
		Title: "Environment Secrets",
		New: func() *cli.Command {
			return &cli.Command{
				Run: func(_ context.Context, _ []string) error {
					if err := s.Load(); err != nil {
						return err
					}

					for _, secret := range s.Storage.Secrets {
						name := strcase.ToScreamingSnake(secret.Name)
						value := secret.Value.String
						value = strings.ReplaceAll(value, "\"", "\\\"")
						value = strings.ReplaceAll(value, "$", "\\$")
						fmt.Printf("%s=\"%s\"\n", name, value)
					}
					return nil
				},
			}
		},
	}
}

func (s *SecretsManager) init() *cli.CommandInfo {
	return &cli.CommandInfo{
		Name:  "init",
		Title: "Generate encryption key",
		New: func() *cli.Command {
			return &cli.Command{
				Run: func(_ context.Context, _ []string) error {
					key := os.Getenv("SECRETSMANAGER_KEY")
					if len(key) != 0 {
						return errors.New("SECRETSMANAGER_KEY already exists")
					}

					g, err := generator.New(&generator.Config{
						Length:                     32,
						IncludeSymbols:             false,
						IncludeNumbers:             true,
						IncludeLowercaseLetters:    true,
						IncludeUppercaseLetters:    true,
						ExcludeSimilarCharacters:   true,
						ExcludeAmbiguousCharacters: true,
					})
					if err != nil {
						return err
					}

					password, err := g.Generate()
					if err != nil {
						return err
					}

					fmt.Println("# Add the following to /etc/environment and store securely in case you need to restore")
					fmt.Println("# WARN: Please, don't add/commit this key to git, as it allows decrypting all secrets.")
					fmt.Println("SECRETSMANAGER_KEY=" + *password)
					return nil
				},
			}
		},
	}
}
