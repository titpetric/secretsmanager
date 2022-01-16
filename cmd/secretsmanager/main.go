package main

import (
	"os"

	"github.com/apex/log"
	"github.com/go-bridget/mig/cli"
)

func start() error {
	manager, err := NewSecretsManager()
	if err != nil {
		return err
	}
	defer manager.Save()

	app := cli.NewApp("secretsmanager")
	for _, command := range manager.Commands() {
		app.AddCommand(command.Name, command.Title, command.New)
	}
	return app.Run()
}

func main() {
	if err := start(); err != nil {
		log.WithError(err).Error("Exiting")
		os.Exit(1)
	}
}
