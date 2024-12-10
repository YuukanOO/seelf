package bus

import (
	"slices"
	"strings"
)

const groupSeparator = "."

// Creates a group identifier from an array of strings representing subjects.
func Group(parts ...string) string {
	slices.Sort(parts)
	return strings.Join(parts, groupSeparator)
}
