package bus_test

import (
	"testing"

	"github.com/YuukanOO/seelf/pkg/assert"
	"github.com/YuukanOO/seelf/pkg/bus"
)

func Test_Topic(t *testing.T) {
	t.Run("should returns a new topic identifier with parts sorted", func(t *testing.T) {
		assert.Equal(t, "bar.foo", bus.Topic("bar", "foo"))
		assert.Equal(t, "bar.foo", bus.Topic("foo", "bar"))
	})
}
