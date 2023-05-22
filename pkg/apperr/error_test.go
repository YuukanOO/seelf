package apperr_test

import (
	"errors"
	"testing"

	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

func Test_Error(t *testing.T) {
	t.Run("could be instantiated with a code", func(t *testing.T) {
		msg := "an error !"
		err := apperr.New(msg)

		testutil.Equals(t, msg, err.Error())
		testutil.ErrorIs(t, apperr.Error{msg, nil}, err)
		testutil.IsTrue(t, errors.As(err, &apperr.Error{}))
	})

	t.Run("could be instantiated with a detail error", func(t *testing.T) {
		err := errors.New("some infrastructure error")
		derr := apperr.NewWithDetail("some_code", err)

		testutil.Equals(t, `some_code:some infrastructure error`, derr.Error())
		testutil.ErrorIs(t, apperr.Error{"some_code", err}, derr)
		testutil.ErrorIs(t, err, derr)
	})

	t.Run("implements the Is function for nested errors", func(t *testing.T) {
		err := apperr.New("some_pouet")
		wrapped := apperr.Wrap(err, errors.New("some infrastructure error"))

		testutil.ErrorIs(t, err, wrapped)
	})
}

func Test_Wrap(t *testing.T) {
	t.Run("should populate the Detail field of a Error", func(t *testing.T) {
		err := apperr.New("some_code")
		detail := errors.New("another error")

		derr := apperr.Wrap(err, detail)

		testutil.Equals(t, `some_code:another error`, derr.Error())
		testutil.ErrorIs(t, apperr.Error{"some_code", detail}, derr)
	})

	t.Run("should create a new Error if err is not one", func(t *testing.T) {
		err := errors.New("some_code")
		detail := errors.New("another error")

		derr := apperr.Wrap(err, detail)
		testutil.Equals(t, `some_code:another error`, derr.Error())
		testutil.ErrorIs(t, apperr.Error{"some_code", detail}, derr)
	})
}

func Test_As(t *testing.T) {
	t.Run("can convert an error to a specific one", func(t *testing.T) {
		err := apperr.New("base app error")

		appErr, ok := apperr.As[apperr.Error](err)

		testutil.IsTrue(t, ok)
		testutil.Equals(t, "base app error", appErr.Error())

		err = errors.New("another one")
		_, ok = apperr.As[apperr.Error](err)
		testutil.IsFalse(t, ok)
	})
}
