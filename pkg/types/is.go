package types

// Checks if the given obj is of type T.
func Is[T any](obj any) bool {
	_, ok := obj.(T)
	return ok
}
