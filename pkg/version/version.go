package version

import (
	"fmt"
	"runtime/debug"
	"slices"
)

const (
	version10 = "1.0"
	version11 = "1.1"
	version12 = "1.2"
)

var SupportedSchemaVersions = []string{version10, version11, version12}

var version string

func GetEibVersion() string {
	if version != "" {
		return version
	}

	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				return fmt.Sprintf("git-%s", setting.Value)
			}
		}
	}

	return "Unknown"
}

func IsSchemaVersionSupported(version string) bool {
	return slices.Contains(SupportedSchemaVersions, version)
}
