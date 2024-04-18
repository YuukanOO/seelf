package validate

import (
	"strings"

	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/monad"
)

var ErrValidationFailed = apperr.New("validation_failed")

type (
	Validator[T any] func(T) error    // Represents a validator for a specific type
	Of               map[string]error // Tiny shorthand to define a map of validators
	FieldErrors      map[string]error // Represents validation errors tied to fields
)

func (e FieldErrors) Error() string {
	var builder strings.Builder

	for name, err := range e {
		builder.WriteString("\n\t" + name + ": " + err.Error())
	}

	return builder.String()
}

// Flatten nested FieldErrors if any and merge the appropriate field names.
// It also removes nil values.
func (e FieldErrors) Flatten() FieldErrors {
	result := make(FieldErrors, len(e))
	flatten(result, e, "")
	return result
}

// Builds a new validation error with given invalid fields.
// Wraps the FieldErrors inside the ErrValidationFailed.
func NewError(fieldErrs FieldErrors) error {
	return apperr.Wrap(ErrValidationFailed, fieldErrs)
}

// Wraps the given error in a new validation error for the specified fields only if
// it is an app level error. If an infrastructure error is given (ie. not an apperr.Error), it will return
// immediately without touching it.
func Wrap(err error, field string, additionalFields ...string) error {
	if err == nil {
		return nil
	}

	if _, isAppErr := apperr.As[apperr.Error](err); !isAppErr {
		return err
	}

	fieldErrs := FieldErrors{field: err}

	for _, f := range additionalFields {
		fieldErrs[f] = err
	}

	return NewError(fieldErrs.Flatten())
}

// Validates a struct by applying the given validators to its fields.
func Struct(definition Of) error {
	fieldErrs := FieldErrors(definition).Flatten()

	if len(fieldErrs) > 0 {
		return NewError(fieldErrs)
	}

	return nil
}

// In many cases, this will be sufficient to apply many validators to one value.
func Field[T any](value T, validators ...Validator[T]) error {
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
	if value, isSet := m.TryGet(); isSet {
		return fn(value)
	}

	return nil
}

// Same as Maybe but for monad.Patch. Executes the function only if the value is set and not nil.
// Looks like I can't pass by an interface to share the logic between Maybe and Patch since
// Go can't infer the T. I prefer to make the usage easier by duplicating the logic in this
// specific case.
// FIXME: When go can infer correctly from an interface [T], merge Maybe and Patch.
func Patch[T any](p monad.Patch[T], fn func(T) error) error {
	if maybeValue, isSet := p.TryGet(); isSet {
		if value, hasValue := maybeValue.TryGet(); hasValue {
			return fn(value)
		}
	}

	return nil
}

func flatten(target FieldErrors, current FieldErrors, prefix string) {
	if prefix != "" {
		prefix = prefix + "."
	}

	for field, err := range current {
		if err == nil {
			continue
		}

		nested, isNested := apperr.As[FieldErrors](err)

		if !isNested {
			target[prefix+field] = err
			continue
		}

		flatten(target, nested, prefix+field)
	}
}
