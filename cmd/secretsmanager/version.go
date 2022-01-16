package main

import (
	"context"

	"github.com/apex/log"
	"github.com/go-bridget/mig/cli"
)

var (
	BuildVersion string
	BuildTime    string

	version = &cli.CommandInfo{
		Name:  "version",
		Title: "Print build version information",
		New: func() *cli.Command {
			return &cli.Command{
				Run: func(_ context.Context, _ []string) error {
					log.Infof("build version %s", BuildVersion)
					log.Infof("build time    %s", BuildTime)
					return nil
				},
			}
		},
	}
)
