package storage

import (
	"encoding/json"
	"strings"
	"unicode/utf8"
)

const (
	secretChar  = "*"
	newlineChar = "\n"
)

// Represents a specific string that should be kept secret and should never be exposed.
// To do that, it implements the MarshalJSON interface to always return a safe representation.
type SecretString string

// Implements the Scan interface to enable the use of this type in a monad.
func (s *SecretString) Scan(src any) error {
	*s = SecretString(src.(string))
	return nil
}

func (s SecretString) MarshalJSON() ([]byte, error) {
	lines := strings.Split(string(s), newlineChar)

	for i, line := range lines {
		lines[i] = strings.Repeat(secretChar, utf8.RuneCountInString(line))
	}

	return json.Marshal(strings.Join(lines, newlineChar))
}
