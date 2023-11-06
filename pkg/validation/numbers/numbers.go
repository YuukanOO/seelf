package numbers

import (
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/validation"
)

var (
	ErrMin = apperr.New("min")
)

func Min(minValue int) validation.Validator[int] {
	return func(value int) error {
		if value < minValue {
			return ErrMin
		}

		return nil
	}
}
