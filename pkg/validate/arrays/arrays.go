package arrays

import "github.com/YuukanOO/seelf/pkg/apperr"

var ErrRequired = apperr.New("required")

func Required[T any](value []T) error {
	if len(value) == 0 {
		return ErrRequired
	}
	return nil
}
