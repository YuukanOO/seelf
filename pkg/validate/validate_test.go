package validate_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/testutil"
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
		testutil.ErrorIs(t, errRequired, err)
	})

	t.Run("returns nil when validation pass successfully", func(t *testing.T) {
		err := validate.Field("something", required)
		testutil.IsNil(t, err)
	})

	t.Run("returns the validator error", func(t *testing.T) {
		err := validate.Field("something", required, alwaysFail)
		testutil.ErrorIs(t, errAlwaysFail, err)
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
		testutil.ErrorIs(t, errRequired, err)
		testutil.Equals(t, "", target)
	})

	t.Run("returns nil error and assign the target upon success", func(t *testing.T) {
		var target objectValue

		err := validate.Value("something", &target, objectValueFactory)
		testutil.IsNil(t, err)
		testutil.Equals(t, "something", target)
	})
}

func Test_Struct(t *testing.T) {
	t.Run("collects validation errors and returns a validation error", func(t *testing.T) {
		err := validate.Struct(validate.Of{
			"firstName": validate.Field("", required),
			"lastName":  validate.Field("doe", required, alwaysFail),
		})

		testutil.Contains(t, "validation_failed:", err.Error())
		testutil.Contains(t, "firstName: required", err.Error())
		testutil.Contains(t, "lastName: always fail", err.Error())
		testutil.ErrorIs(t, validate.ErrValidationFailed, err)
		validationErr, ok := apperr.As[validate.FieldErrors](err)
		testutil.IsTrue(t, ok)
		testutil.Equals(t, 2, len(validationErr))
		testutil.ErrorIs(t, errRequired, validationErr["firstName"])
		testutil.ErrorIs(t, errAlwaysFail, validationErr["lastName"])
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
		testutil.IsTrue(t, ok)
		testutil.Equals(t, 4, len(validationErr))
		testutil.ErrorIs(t, errRequired, validationErr["firstName"])
		testutil.ErrorIs(t, errAlwaysFail, validationErr["lastName"])
		testutil.ErrorIs(t, errRequired, validationErr["nested.firstName"])
		testutil.ErrorIs(t, errRequired, validationErr["nested.nested.firstName"])
	})

	t.Run("returns nil if no error exists", func(t *testing.T) {
		err := validate.Struct(validate.Of{
			"firstName": validate.Field("john", required),
			"lastName":  validate.Field("doe", required),
		})

		testutil.IsNil(t, err)
	})
}

func Test_If(t *testing.T) {
	t.Run("return the validation error only if expression is true", func(t *testing.T) {
		err := validate.Struct(validate.Of{
			"firstname": validate.If(false, func() error { return validate.Field("", required) }),
			"lastName":  validate.If(true, func() error { return validate.Field("", required) }),
		})

		testutil.Equals(t, `validation_failed:
	lastName: required`, err.Error())
		validationErr, ok := apperr.As[validate.FieldErrors](err)
		testutil.IsTrue(t, ok)
		testutil.Equals(t, 1, len(validationErr))
		testutil.ErrorIs(t, errRequired, validationErr["lastName"])
	})
}

func Test_Maybe(t *testing.T) {
	t.Run("does not execute the function if the monad is not set", func(t *testing.T) {
		var m monad.Maybe[string]

		err := validate.Maybe(m, func(val string) error {
			return validate.Field(val, required)
		})

		testutil.IsNil(t, err)
	})

	t.Run("executes the function if the monad is set", func(t *testing.T) {
		m := monad.Value("")

		err := validate.Maybe(m, func(val string) error {
			return validate.Field(val, required)
		})

		testutil.ErrorIs(t, errRequired, err)
	})
}

func Test_Patch(t *testing.T) {
	t.Run("does not execute the function if the patch is not set", func(t *testing.T) {
		var p monad.Patch[string]

		err := validate.Patch(p, func(val string) error {
			return validate.Field(val, required)
		})

		testutil.IsNil(t, err)
	})

	t.Run("executes the function if the patch is set", func(t *testing.T) {
		p := monad.PatchValue("")

		err := validate.Patch(p, func(val string) error {
			return validate.Field(val, required)
		})

		testutil.ErrorIs(t, errRequired, err)
	})
}

func Test_Wrap(t *testing.T) {
	t.Run("returns the error if it's not an application level error", func(t *testing.T) {
		infrastructureErr := errors.New("an infrastructure error")

		testutil.IsTrue(t, validate.Wrap(infrastructureErr, "one", "two") == infrastructureErr)
		testutil.IsTrue(t, validate.Wrap(nil, "one", "two") == nil)
	})

	t.Run("returns nil if no err is given", func(t *testing.T) {
		testutil.IsNil(t, validate.Wrap(nil, "one", "two"))
	})

	t.Run("wrap the application error for the specified fields", func(t *testing.T) {
		appErr := apperr.New("application level error")
		err := validate.Wrap(appErr, "one", "two")

		testutil.ErrorIs(t, validate.ErrValidationFailed, err)

		validationErr, ok := apperr.As[validate.FieldErrors](err)
		fmt.Println(validationErr.Error())
		testutil.IsTrue(t, ok)
		testutil.Equals(t, 2, len(validationErr))
		testutil.ErrorIs(t, appErr, validationErr["one"])
		testutil.ErrorIs(t, appErr, validationErr["two"])
	})

	t.Run("flatten nested validation errors", func(t *testing.T) {
		appErr := validate.Struct(validate.Of{
			"firstName": validate.Field("", required),
			"lastName":  validate.Field("", alwaysFail),
		})

		err := validate.Wrap(appErr, "one", "two")

		testutil.ErrorIs(t, validate.ErrValidationFailed, err)
		validationErr, ok := apperr.As[validate.FieldErrors](err)
		testutil.IsTrue(t, ok)
		testutil.Equals(t, 4, len(validationErr))
		testutil.ErrorIs(t, errRequired, validationErr["one.firstName"])
		testutil.ErrorIs(t, errAlwaysFail, validationErr["one.lastName"])
		testutil.ErrorIs(t, errRequired, validationErr["two.firstName"])
		testutil.ErrorIs(t, errAlwaysFail, validationErr["two.lastName"])
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

		testutil.DeepEquals(t, validate.FieldErrors{
			"1":   errRequired,
			"2.1": errAlwaysFail,
			"3.1": errRequired,
		}, flattened)
	})
}
