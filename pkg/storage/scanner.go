package storage

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

var ErrCouldNotUnmarshalGivenType = errors.New("could not unmarshal given type")

type (
	// Deserialize a storage row to field. This is the interface making possible the
	// rehydration of domain entities from the db since all fields are private to enforce
	// invariants and encapsulation.
	//
	// Since domain store will always constructs an aggregate as a whole, it makes the process
	// relatively easy to keep under control.
	Scanner interface {
		// Scan current row into the destination fields.
		// The things to keep in mind is the order used when scanning which should always be the
		// same as the order of fields returned by the underlying implementation (for a database,
		// the order of SELECT columns).
		//
		// IMPORTANT: it will fails if the type of a monad.Value is not a primitive type or
		// does not implements the sql.Scanner interface.
		Scan(...any) error
	}

	Mapper[T any]                  func(Scanner) (T, error)         // Mapper function from a simple Scanner to an object of type T
	Merger[TParent, TChildren any] func(TParent, TChildren) TParent // Merger function to handle relations by configuring how a child should be merged into its parent
)

// Ease the scan of a json serialized field.
func ScanJSON[T any](value any, target *T) error {
	str, asStr := value.(string)

	if !asStr {
		return ErrCouldNotUnmarshalGivenType
	}

	return json.Unmarshal([]byte(str), target)
}

// Ease the valueing of a json serialized field by calling json.Marshal and returning
// a string as accepted as a valid driver.Value.
func ValueJSON[T any](v T) (driver.Value, error) {
	b, err := json.Marshal(v)

	return string(b), err
}
