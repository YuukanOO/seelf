package query

import (
	"encoding/json"
)

// Some information must be kept secure (such as the repository access token) and should
// never be exposed in the API.
const secretStringPublicValue = "<unexposed>"

// Represents a specific string that should be kept secret and should never be exposed.
// To do that, it implements the MarshalJSON interface to always return the same constant (See SecretStringPublicValue).
//
// Maybe this type should be moved to a shared/app package.
type SecretString string

// Implements the Scan interface to enable the use of this type in a monad.
func (s *SecretString) Scan(src any) error {
	*s = SecretString(src.(string))
	return nil
}

func (s SecretString) MarshalJSON() ([]byte, error) {
	return json.Marshal(secretStringPublicValue)
}
