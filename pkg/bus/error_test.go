package bus_test

import (
	"errors"
	"testing"

	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

func Test_Error(t *testing.T) {
	t.Run("should correctly unwrap the inner error", func(t *testing.T) {
		innerErr := errors.New("some error")

		err := bus.Ignore(innerErr)

		testutil.ErrorIs(t, innerErr, err)
	})

	t.Run("should returns nil when trying to build an error from a nil one", func(t *testing.T) {
		testutil.IsNil(t, bus.Ignore(nil))
		testutil.IsNil(t, bus.PreserveOrder(nil))
	})
}
