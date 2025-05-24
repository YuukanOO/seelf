package bus

import (
	"strings"
)

const keySeparator = "."

// Scheduled job key
type JobKey string

// Build a job key from different parts
func JobKeyFrom(parts ...string) JobKey {
	return JobKey(strings.Join(parts, keySeparator))
}
