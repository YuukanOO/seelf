package bus_test

import (
	"testing"

	"github.com/YuukanOO/seelf/pkg/assert"
	"github.com/YuukanOO/seelf/pkg/bus"
)

func Test_Group(t *testing.T) {
	t.Run("should returns a new group identifier with parts sorted", func(t *testing.T) {
		assert.Equal(t, "bar.foo", bus.Group("bar", "foo"))
		assert.Equal(t, "bar.foo", bus.Group("foo", "bar"))
	})
}
