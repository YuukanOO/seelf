package bus_test

import (
	"testing"

	"github.com/YuukanOO/seelf/pkg/assert"
	"github.com/YuukanOO/seelf/pkg/bus"
)

func Test_JobKey(t *testing.T) {
	t.Run("should returns a new job key from given parts", func(t *testing.T) {
		assert.Equal(t, "bar.foo", bus.JobKeyFrom("bar", "foo"))
		assert.Equal(t, "foo.bar", bus.JobKeyFrom("foo", "bar"))
	})
}
