package validation_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/testutil"
	"github.com/YuukanOO/seelf/pkg/validation"
)

var (
	errRequired   = errors.New("required")
	errAlwaysFail = errors.New("always fail")
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

func Test_Is(t *testing.T) {
	t.Run("call every validators", func(t *testing.T) {
		err := validation.Is("", required, alwaysFail)
		testutil.ErrorIs(t, errRequired, err)
	})

	t.Run("returns nil when validation pass successfuly", func(t *testing.T) {
		err := validation.Is("something", required)
		testutil.IsNil(t, err)
	})

	t.Run("returns the validator error", func(t *testing.T) {
		err := validation.Is("something", required, alwaysFail)
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

		err := validation.Value("", &target, objectValueFactory)
		testutil.ErrorIs(t, errRequired, err)
		testutil.Equals(t, "", target)
	})

	t.Run("returns nil error and assign the target upon success", func(t *testing.T) {
		var target objectValue

		err := validation.Value("something", &target, objectValueFactory)
		testutil.IsNil(t, err)
		testutil.Equals(t, "something", target)
	})
}

func Test_Check(t *testing.T) {
	t.Run("collects validation errors and returns a validation error", func(t *testing.T) {
		err := validation.Check(validation.Of{
			"firstName": validation.Is("", required),
			"lastName":  validation.Is("doe", required, alwaysFail),
		})

		testutil.Contains(t, "validation_failed:", err.Error())
		testutil.Contains(t, "firstName: required", err.Error())
		testutil.Contains(t, "lastName: always fail", err.Error())
		testutil.ErrorIs(t, validation.ErrValidationFailed, err)
		validationErr, ok := apperr.As[validation.Error](err)
		testutil.IsTrue(t, ok)
		testutil.Equals(t, 2, len(validationErr.Fields))
		testutil.ErrorIs(t, errRequired, validationErr.Fields["firstName"])
		testutil.ErrorIs(t, errAlwaysFail, validationErr.Fields["lastName"])
	})

	t.Run("returns nil if no error exists", func(t *testing.T) {
		err := validation.Check(validation.Of{
			"firstName": validation.Is("john", required),
			"lastName":  validation.Is("doe", required),
		})

		testutil.IsNil(t, err)
	})
}

func Test_If(t *testing.T) {
	t.Run("return the validation error only if expression is true", func(t *testing.T) {
		err := validation.Check(validation.Of{
			"firstname": validation.If(false, func() error { return validation.Is("", required) }),
			"lastName":  validation.If(true, func() error { return validation.Is("", required) }),
		})

		testutil.Equals(t, `validation_failed:
	lastName: required`, err.Error())
		validationErr, ok := apperr.As[validation.Error](err)
		testutil.IsTrue(t, ok)
		testutil.Equals(t, 1, len(validationErr.Fields))
		testutil.ErrorIs(t, errRequired, validationErr.Fields["lastName"])
	})
}

func Test_Maybe(t *testing.T) {
	t.Run("does not execute the function if the monad is not set", func(t *testing.T) {
		var m monad.Maybe[string]

		err := validation.Maybe(m, func(val string) error {
			return validation.Is(val, required)
		})

		testutil.IsNil(t, err)
	})

	t.Run("executes the function if the monad is set", func(t *testing.T) {
		m := monad.Value("")

		err := validation.Maybe(m, func(val string) error {
			return validation.Is(val, required)
		})

		testutil.ErrorIs(t, errRequired, err)
	})
}

func Test_Patch(t *testing.T) {
	t.Run("does not execute the function if the patch is not set", func(t *testing.T) {
		var p monad.Patch[string]

		err := validation.Patch(p, func(val string) error {
			return validation.Is(val, required)
		})

		testutil.IsNil(t, err)
	})

	t.Run("executes the function if the patch is set", func(t *testing.T) {
		p := monad.PatchValue("")

		err := validation.Patch(p, func(val string) error {
			return validation.Is(val, required)
		})

		testutil.ErrorIs(t, errRequired, err)
	})
}

func Test_WrapIfAppErr(t *testing.T) {
	t.Run("returns the error if it's not an application level error", func(t *testing.T) {
		infrastructureErr := errors.New("an infrastructure error")

		testutil.IsTrue(t, validation.WrapIfAppErr(infrastructureErr, "one", "two") == infrastructureErr)
		testutil.IsTrue(t, validation.WrapIfAppErr(nil, "one", "two") == nil)
	})

	t.Run("wrap the application error for the specified fields", func(t *testing.T) {
		appErr := apperr.New("application level error")
		err := validation.WrapIfAppErr(appErr, "one", "two")

		testutil.ErrorIs(t, validation.ErrValidationFailed, err)

		validationErr, ok := apperr.As[validation.Error](err)
		fmt.Println(validationErr.Error())
		testutil.IsTrue(t, ok)
		testutil.Equals(t, 2, len(validationErr.Fields))
		testutil.ErrorIs(t, appErr, validationErr.Fields["one"])
		testutil.ErrorIs(t, appErr, validationErr.Fields["two"])
	})
}
