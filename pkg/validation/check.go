package validation

import (
	"fmt"
	"strings"

	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/monad"
)

var ErrValidationFailed = apperr.New("validation_failed")

type (
	Validator[T any] func(T) error    // Represents a validator for a specific type
	Of               map[string]error // Tiny shorthand to define a map of validators

	// Validation errors struct containing field related errors
	Error struct {
		Fields map[string]error `json:"fields"`
	}
)

func (e Error) Error() string {
	var builder strings.Builder

	for name, err := range e.Fields {
		builder.WriteString(fmt.Sprintf("\n\t%s: %s", name, err))
	}

	return builder.String()
}

// Builds a new validation error with given invalid fields
func NewError(fieldErrs map[string]error) error {
	return apperr.Wrap(ErrValidationFailed, Error{fieldErrs})
}

// Wraps the given error in a new validation error for the specified fields only if
// it is an app level error. If an infrastructure error is given, it will return
// immediately without touching it.
func WrapIfAppErr(err error, field string, additionalFields ...string) error {
	if _, isAppErr := apperr.As[apperr.Error](err); !isAppErr {
		return err
	}

	fieldErrs := map[string]error{field: err}

	for _, f := range additionalFields {
		fieldErrs[f] = err
	}

	return NewError(fieldErrs)
}

// Checks a map of label / errors and returns a validation error which contains
// every errors as needed.
func Check(definition Of) error {
	fieldErrs := make(map[string]error)

	for f, err := range definition {
		if err != nil {
			fieldErrs[f] = err
		}
	}

	if len(fieldErrs) > 0 {
		return NewError(fieldErrs)
	}

	return nil
}

// In many cases, this will be sufficient to apply many validators to one value.
func Is[T any](value T, validators ...Validator[T]) error {
	for _, validator := range validators {
		if err := validator(value); err != nil {
			return err
		}
	}

	return nil
}

// Validate object values by calling their factory and writing to the target in
// the same call. It makes it easy to validates and instantiates with one call.
func Value[TRaw, TTarget any](value TRaw, target *TTarget, factory func(TRaw) (TTarget, error)) error {
	res, err := factory(value)

	if err != nil {
		return err
	}

	*target = res

	return nil
}

// Simple function to returns the validation result only if the given expr is true.
func If(expr bool, fn func() error) error {
	if expr {
		return fn()
	}

	return nil
}

// Same as If but executes the fn if the monad has a value.
func Maybe[T any](m monad.Maybe[T], fn func(T) error) error {
	if m.HasValue() {
		return fn(m.MustGet())
	}

	return nil
}

// Same as Maybe but for monad.Patch. Executes the function only if the value is set and not nil.
// Looks like I can't pass by an interface to share the logic between Maybe and Patch since
// Go can't infer the T. I prefer to make the usage easier by duplicating the logic in this
// specific case.
// FIXME: When go can infer correctly from an interface [T], merge Maybe and Patch.
func Patch[T any](p monad.Patch[T], fn func(T) error) error {
	if p.HasValue() {
		return fn(p.MustGet())
	}

	return nil
}
