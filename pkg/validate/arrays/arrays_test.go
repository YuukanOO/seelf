package arrays_test

import (
	"testing"

	"github.com/YuukanOO/seelf/pkg/assert"
	"github.com/YuukanOO/seelf/pkg/validate/arrays"
)

func Test_Required(t *testing.T) {
	t.Run("should fail on empty arrays", func(t *testing.T) {
		assert.ErrorIs(t, arrays.ErrRequired, arrays.Required([]string{}))
		assert.ErrorIs(t, arrays.ErrRequired, arrays.Required[string](nil))
	})

	t.Run("should succeed on non-empty arrays", func(t *testing.T) {
		assert.Nil(t, arrays.Required([]string{"good!"}))
	})
}
