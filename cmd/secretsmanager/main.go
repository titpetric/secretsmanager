package main

import (
	"os"

	"github.com/titpetric/cli"

	"github.com/titpetric/secretsmanager/internal/logger"
)

// start builds the cli and runs the command named on the command line.
func start(log Logger) error {
	manager, err := NewSecretsManager(log)
	if err != nil {
		return err
	}

	app := cli.NewApp("secretsmanager")
	for _, command := range manager.Commands() {
		app.AddCommand(command.Name, command.Title, command.New)
	}

	// Commands which change a secret save it themselves, with the context
	// they were given.
	return app.Run()
}

// main runs the cli, and exits non-zero if the command failed.
func main() {
	log := logger.New(os.Stderr)

	if err := start(log); err != nil {
		log.Error("exiting", "error", err)
		os.Exit(1)
	}
}
