package main

import (
	"context"
	"runtime/debug"

	"github.com/titpetric/cli"
)

// BuildVersion and BuildTime are set with -ldflags at build time.
var (
	BuildVersion string
	BuildTime    string
)

// version returns the command which prints the build information.
func (s *SecretsManager) version() *cli.CommandInfo {
	return &cli.CommandInfo{
		Name:  "version",
		Title: "Print build version information",
		New: func() *cli.Command {
			return &cli.Command{
				Run: func(_ context.Context, _ []string) error {
					buildVersion, buildTime := buildInfo()
					s.log.Info("build", "version", buildVersion, "time", buildTime)
					return nil
				},
			}
		},
	}
}

// buildInfo returns the build version and time. Without the ldflags which
// set them it falls back to the revision the go tool records from git.
func buildInfo() (buildVersion, buildTime string) {
	buildVersion, buildTime = BuildVersion, BuildTime
	if buildVersion != "" && buildTime != "" {
		return buildVersion, buildTime
	}

	info, ok := debug.ReadBuildInfo()
	if !ok {
		return buildVersion, buildTime
	}

	return buildInfoFrom(info.Settings, buildVersion, buildTime)
}

// buildInfoFrom fills in whatever ldflags didn't set from the settings the
// go tool recorded, and leaves the rest empty.
func buildInfoFrom(settings []debug.BuildSetting, buildVersion, buildTime string) (string, string) {
	for _, setting := range settings {
		switch setting.Key {
		case "vcs.revision":
			if buildVersion == "" {
				buildVersion = setting.Value
			}
		case "vcs.time":
			if buildTime == "" {
				buildTime = setting.Value
			}
		}
	}
	return buildVersion, buildTime
}
