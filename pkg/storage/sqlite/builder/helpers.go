package builder

import "github.com/YuukanOO/seelf/pkg/storage"

// Tiny mapper when all you have to do is retrieve a single value from a query.
func valueMapper[T any](scanner storage.Scanner) (value T, err error) {
	err = scanner.Scan(&value)
	return value, err
}
