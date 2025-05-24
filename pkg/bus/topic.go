package bus

import (
	"slices"
	"strings"
)

const topicSeparator = "."

// Creates a topic identifier from an array of strings representing subjects.
func Topic(parts ...string) string {
	slices.Sort(parts)
	return strings.Join(parts, topicSeparator)
}
