package must_test

import (
	"errors"
	"testing"

	"github.com/YuukanOO/seelf/pkg/assert"
	"github.com/YuukanOO/seelf/pkg/must"
)

func Test_Panic(t *testing.T) {
	t.Run("should panic if an error is given", func(t *testing.T) {
		err := errors.New("some error")
		defer func() {
			r := recover()

			assert.NotNil(t, r)
			assert.ErrorIs(t, err, r.(error))
		}()

		must.Panic(42, err)
	})

	t.Run("should return the value if no error is given", func(t *testing.T) {
		value := must.Panic(42, nil)

		assert.Equal(t, 42, value)
	})
}
