package id

import "github.com/segmentio/ksuid"

// Generates a new random unique identifier.
func New[T ~string]() T {
	return T(ksuid.New().String())
}
