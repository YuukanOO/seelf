package version

import (
	"runtime/debug"
)

var version = "2.0.0"

// Retrieve the currentVersion version with additional vcs info if any.
func Current() string {
	var suffix string

	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				suffix = "-" + setting.Value
				break
			}
		}
	}

	return version + suffix
}
