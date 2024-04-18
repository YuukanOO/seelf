package monad

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"

	"gopkg.in/yaml.v3"
)

// Represents an optional value of type T. It implements some infrastructure interfaces
// such as JSON (un)marshalling and database convert.
//
// This is much more clean than using pointers to express optional values.
type Maybe[T any] struct {
	hasValue bool
	value    T
}

// Instantiates a monad with a defined value.
func Value[T any](value T) (m Maybe[T]) {
	m.Set(value)
	return m
}

// Instantiates an empty monad for the given type.
func None[T any]() (m Maybe[T]) {
	return m
}

// Assign the given value to the monad.
func (m *Maybe[T]) Set(value T) {
	m.hasValue = true
	m.value = value
}

// Unset this monad value.
func (m *Maybe[T]) Unset() {
	m.hasValue = false
}

// Has a value been set on this monad?
func (m Maybe[T]) HasValue() bool { return m.hasValue }

// Retrieve the inner value and panic if none is set.
func (m Maybe[T]) MustGet() T {
	if !m.hasValue {
		panic("trying to access a monad's value but none is set")
	}

	return m.value
}

// Get the inner value and a boolean indicating if it has been set.
func (m Maybe[T]) TryGet() (T, bool) {
	return m.value, m.hasValue
}

// Retrieve the inner value or the fallback if it doesn't have one.
func (m Maybe[T]) Get(fallback T) T {
	if !m.hasValue {
		return fallback
	}

	return m.value
}

// Implements the db valuer interface to persist it easily. If you want the value of
// the maybe, you may check Get instead.
func (m Maybe[T]) Value() (driver.Value, error) {
	if m.hasValue {
		// Will check if the m.value implements the Valuer interface so no problem here
		return driver.DefaultParameterConverter.ConvertValue(m.value)
	}

	return nil, nil
}

// Implements the db scanner interface to retrieve it easily from the storage.
func (m *Maybe[T]) Scan(value any) error {
	if value == nil {
		return nil
	}

	// Retrieve value is probably a primitive one so check first if the value hold
	// by this monad implements the sql.Scanner interface, if so, call it.
	// If not, it will use the default parameter converter.
	switch converted := any(&m.value).(type) {
	case sql.Scanner:
		if err := converted.Scan(value); err != nil {
			return err
		}
	default:
		v, err := driver.DefaultParameterConverter.ConvertValue(value)

		if err != nil {
			return err
		}

		m.value = v.(T) // FIXME: it will panic if T is not the same type as v (for example, when scanning optional domain.UserID)
	}

	// Either way, no error, it means the value has been retrieved correctly.
	m.hasValue = true

	return nil
}

func (m Maybe[T]) MarshalJSON() ([]byte, error) {
	if m.hasValue {
		return json.Marshal(m.value)
	}

	return json.Marshal(nil)
}

func (m *Maybe[T]) UnmarshalJSON(data []byte) error {
	var target *T

	if err := json.Unmarshal(data, &target); err != nil {
		return err
	}

	// Handle the literal string "null" which maps to a nil pointer so that's because
	// the monad doesn't have a value.
	if target == nil {
		m.hasValue = false
		return nil
	}

	m.hasValue = true
	m.value = *target

	return nil
}

func (m Maybe[T]) MarshalYAML() (any, error) {
	if m.hasValue {
		return m.value, nil
	}

	return nil, nil
}

func (m Maybe[T]) IsZero() bool { return !m.hasValue }

func (m *Maybe[T]) UnmarshalYAML(value *yaml.Node) error {
	// Values sets explicitly in a yaml file should have the correct type
	if err := value.Decode(&m.value); err != nil {
		return err
	}

	m.hasValue = true

	return nil
}

func (m *Maybe[T]) UnmarshalEnvironmentValue(data string) error {
	// If the value is a string, it should be quoted for the json.Unmarshal to work correctly
	if _, isString := any(&m.value).(*string); data != "" && isString {
		data = `"` + data + `"`
	}

	if err := json.Unmarshal([]byte(data), &m.value); err != nil {
		m.hasValue = false
		return nil
	}

	m.hasValue = true

	return nil
}
