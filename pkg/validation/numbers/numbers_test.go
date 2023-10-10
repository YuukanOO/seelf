package numbers_test

import (
	"testing"

	"github.com/YuukanOO/seelf/pkg/testutil"
	"github.com/YuukanOO/seelf/pkg/validation/numbers"
)

func Test_Min(t *testing.T) {
	t.Run("should fail on value lesser than the required min", func(t *testing.T) {
		testutil.ErrorIs(t, numbers.ErrMin, numbers.Min(3)(2))
		testutil.ErrorIs(t, numbers.ErrMin, numbers.Min(3)(1))
	})

	t.Run("should succeed on value greater then the required min", func(t *testing.T) {
		testutil.IsNil(t, numbers.Min(3)(4))
		testutil.IsNil(t, numbers.Min(3)(3))
	})
}
