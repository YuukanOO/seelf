package cmd

import (
	"fmt"
	"runtime/debug"
)

var version = "1.0.1"

// Retrieve the currentVersion version with additional vcs info if any.
func currentVersion() string {
	var suffix string

	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				suffix = fmt.Sprintf("-%s", setting.Value)
				break
			}
		}
	}

	return version + suffix
}
