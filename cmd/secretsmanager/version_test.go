package main

import (
	"runtime/debug"
	"testing"
)

// TestBuildInfo covers the build information the version command prints.
func TestBuildInfo(t *testing.T) {
	saved, savedTime := BuildVersion, BuildTime
	t.Cleanup(func() { BuildVersion, BuildTime = saved, savedTime })

	t.Run("from ldflags", func(t *testing.T) {
		BuildVersion, BuildTime = "v1.2.3", "2026-08-23T00:00:00Z"

		version, buildTime := buildInfo()
		if version != "v1.2.3" || buildTime != "2026-08-23T00:00:00Z" {
			t.Errorf("buildInfo() = %q, %q, want the values set with ldflags", version, buildTime)
		}
	})

	// Without ldflags the go tool's own record of the revision is used. A
	// test binary may be built without one, so only the fallback is checked.
	t.Run("from the build info", func(t *testing.T) {
		BuildVersion, BuildTime = "", ""

		version, buildTime := buildInfo()
		if version == "v1.2.3" || buildTime == "2026-08-23T00:00:00Z" {
			t.Error("buildInfo() returned the ldflags values after they were cleared")
		}
	})

	t.Run("from the vcs settings", func(t *testing.T) {
		settings := []debug.BuildSetting{
			{Key: "-buildmode", Value: "exe"},
			{Key: "vcs.revision", Value: "139d40bb45877f868da7767faabf848b559f13fc"},
			{Key: "vcs.time", Value: "2026-08-20T12:46:53Z"},
		}

		version, buildTime := buildInfoFrom(settings, "", "")
		if version != "139d40bb45877f868da7767faabf848b559f13fc" {
			t.Errorf("version = %q, want the revision", version)
		}
		if buildTime != "2026-08-20T12:46:53Z" {
			t.Errorf("buildTime = %q, want the commit time", buildTime)
		}
	})

	// What ldflags set wins over what the go tool recorded.
	t.Run("ldflags win over the vcs settings", func(t *testing.T) {
		settings := []debug.BuildSetting{
			{Key: "vcs.revision", Value: "139d40bb"},
			{Key: "vcs.time", Value: "2026-08-20T12:46:53Z"},
		}

		version, buildTime := buildInfoFrom(settings, "v1.2.3", "")
		if version != "v1.2.3" {
			t.Errorf("version = %q, want the value from ldflags", version)
		}
		if buildTime != "2026-08-20T12:46:53Z" {
			t.Errorf("buildTime = %q, want the commit time", buildTime)
		}
	})

	t.Run("no vcs settings", func(t *testing.T) {
		version, buildTime := buildInfoFrom(nil, "", "")
		if version != "" || buildTime != "" {
			t.Errorf("buildInfoFrom() = %q, %q, want both empty", version, buildTime)
		}
	})
}
