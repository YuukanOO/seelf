package validate_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/assert"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/validate"
)

var (
	errRequired   = apperr.New("required")
	errAlwaysFail = apperr.New("always fail")
)

func required(value string) error {
	if value == "" {
		return errRequired
	}
	return nil
}

func alwaysFail(value string) error {
	return errAlwaysFail
}

func Test_Field(t *testing.T) {
	t.Run("call every validators", func(t *testing.T) {
		err := validate.Field("", required, alwaysFail)
		assert.ErrorIs(t, errRequired, err)
	})

	t.Run("returns nil when validation pass successfully", func(t *testing.T) {
		err := validate.Field("something", required)
		assert.Nil(t, err)
	})

	t.Run("returns the validator error", func(t *testing.T) {
		err := validate.Field("something", required, alwaysFail)
		assert.ErrorIs(t, errAlwaysFail, err)
	})
}

type objectValue string

func objectValueFactory(v string) (objectValue, error) {
	if v == "" {
		return objectValue(""), errRequired
	}

	return objectValue(v), nil
}

func Test_Value(t *testing.T) {
	t.Run("returns error of the factory and doesn't assign the value if it fails", func(t *testing.T) {
		var target objectValue

		err := validate.Value("", &target, objectValueFactory)
		assert.ErrorIs(t, errRequired, err)
		assert.Equal(t, "", target)
	})

	t.Run("returns nil error and assign the target upon success", func(t *testing.T) {
		var target objectValue

		err := validate.Value("something", &target, objectValueFactory)
		assert.Nil(t, err)
		assert.Equal(t, "something", target)
	})
}

func Test_Struct(t *testing.T) {
	t.Run("collects validation errors and returns a validation error", func(t *testing.T) {
		err := validate.Struct(validate.Of{
			"firstName": validate.Field("", required),
			"lastName":  validate.Field("doe", required, alwaysFail),
		})

		assert.Match(t, "validation_failed:", err.Error())
		assert.Match(t, "firstName: required", err.Error())
		assert.Match(t, "lastName: always fail", err.Error())
		assert.ValidationError(t, validate.FieldErrors{
			"firstName": errRequired,
			"lastName":  errAlwaysFail,
		}, err)
	})

	t.Run("merge nested validation errors", func(t *testing.T) {
		err := validate.Struct(validate.Of{
			"firstName": validate.Field("", required),
			"lastName":  validate.Field("doe", required, alwaysFail),
			"nested": validate.Struct(validate.Of{
				"firstName": validate.Field("", required),
				"nested": validate.Struct(validate.Of{
					"firstName": validate.Field("", required),
				}),
			}),
		})

		validationErr, ok := apperr.As[validate.FieldErrors](err)
		assert.True(t, ok)
		assert.Equal(t, 4, len(validationErr))
		assert.ErrorIs(t, errRequired, validationErr["firstName"])
		assert.ErrorIs(t, errAlwaysFail, validationErr["lastName"])
		assert.ErrorIs(t, errRequired, validationErr["nested.firstName"])
		assert.ErrorIs(t, errRequired, validationErr["nested.nested.firstName"])
	})

	t.Run("returns nil if no error exists", func(t *testing.T) {
		err := validate.Struct(validate.Of{
			"firstName": validate.Field("john", required),
			"lastName":  validate.Field("doe", required),
		})

		assert.Nil(t, err)
	})
}

func Test_Array(t *testing.T) {
	t.Run("collects validation errors and returns field errors", func(t *testing.T) {
		err := validate.Array([]string{"john", "doe"}, func(value string, index int) error {
			if value == "doe" {
				return errAlwaysFail
			}

			return nil
		})

		validationErr, ok := apperr.As[validate.FieldErrors](err)

		assert.True(t, ok)
		assert.DeepEqual(t, validate.FieldErrors{
			"1": errAlwaysFail,
		}, validationErr)
	})

	t.Run("returns nil if no error exists", func(t *testing.T) {
		err := validate.Array([]string{"john", "doe"}, func(value string, index int) error {
			return nil
		})

		assert.Nil(t, err)
	})
}

func Test_If(t *testing.T) {
	t.Run("return the validation error only if expression is true", func(t *testing.T) {
		err := validate.Struct(validate.Of{
			"firstname": validate.If(false, func() error { return validate.Field("", required) }),
			"lastName":  validate.If(true, func() error { return validate.Field("", required) }),
		})

		assert.Equal(t, `validation_failed:
	lastName: required`, err.Error())
		validationErr, ok := apperr.As[validate.FieldErrors](err)
		assert.True(t, ok)
		assert.Equal(t, 1, len(validationErr))
		assert.ErrorIs(t, errRequired, validationErr["lastName"])
	})
}

func Test_Maybe(t *testing.T) {
	t.Run("does not execute the function if the monad is not set", func(t *testing.T) {
		var m monad.Maybe[string]

		err := validate.Maybe(m, func(val string) error {
			return validate.Field(val, required)
		})

		assert.Nil(t, err)
	})

	t.Run("executes the function if the monad is set", func(t *testing.T) {
		m := monad.Value("")

		err := validate.Maybe(m, func(val string) error {
			return validate.Field(val, required)
		})

		assert.ErrorIs(t, errRequired, err)
	})
}

func Test_Patch(t *testing.T) {
	t.Run("does not execute the function if the patch is not set", func(t *testing.T) {
		var p monad.Patch[string]

		err := validate.Patch(p, func(val string) error {
			return validate.Field(val, required)
		})

		assert.Nil(t, err)
	})

	t.Run("executes the function if the patch is set", func(t *testing.T) {
		p := monad.PatchValue("")

		err := validate.Patch(p, func(val string) error {
			return validate.Field(val, required)
		})

		assert.ErrorIs(t, errRequired, err)
	})
}

func Test_Wrap(t *testing.T) {
	t.Run("returns the error if it's not an application level error", func(t *testing.T) {
		infrastructureErr := errors.New("an infrastructure error")

		assert.True(t, validate.Wrap(infrastructureErr, "one", "two") == infrastructureErr)
		assert.True(t, validate.Wrap(nil, "one", "two") == nil)
	})

	t.Run("returns nil if no err is given", func(t *testing.T) {
		assert.Nil(t, validate.Wrap(nil, "one", "two"))
	})

	t.Run("wrap the application error for the specified fields", func(t *testing.T) {
		appErr := apperr.New("application level error")
		err := validate.Wrap(appErr, "one", "two")

		assert.ErrorIs(t, validate.ErrValidationFailed, err)

		validationErr, ok := apperr.As[validate.FieldErrors](err)
		fmt.Println(validationErr.Error())
		assert.True(t, ok)
		assert.Equal(t, 2, len(validationErr))
		assert.ErrorIs(t, appErr, validationErr["one"])
		assert.ErrorIs(t, appErr, validationErr["two"])
	})

	t.Run("flatten nested validation errors", func(t *testing.T) {
		appErr := validate.Struct(validate.Of{
			"firstName": validate.Field("", required),
			"lastName":  validate.Field("", alwaysFail),
		})

		err := validate.Wrap(appErr, "one", "two")

		assert.ErrorIs(t, validate.ErrValidationFailed, err)
		validationErr, ok := apperr.As[validate.FieldErrors](err)
		assert.True(t, ok)
		assert.Equal(t, 4, len(validationErr))
		assert.ErrorIs(t, errRequired, validationErr["one.firstName"])
		assert.ErrorIs(t, errAlwaysFail, validationErr["one.lastName"])
		assert.ErrorIs(t, errRequired, validationErr["two.firstName"])
		assert.ErrorIs(t, errAlwaysFail, validationErr["two.lastName"])
	})
}

func Test_FieldErrors(t *testing.T) {
	t.Run("could be flattened", func(t *testing.T) {
		flattened := validate.FieldErrors{
			"1": errRequired,
			"2": validate.FieldErrors{
				"1": errAlwaysFail,
			},
			"3": validate.NewError(validate.FieldErrors{
				"1": errRequired,
			}),
			"4": nil,
		}.Flatten()

		assert.DeepEqual(t, validate.FieldErrors{
			"1":   errRequired,
			"2.1": errAlwaysFail,
			"3.1": errRequired,
		}, flattened)
	})
}
