package version

import (
	"fmt"
	"runtime/debug"
)

var version string

func GetVersion() string {
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
