package bus_test

import (
	"testing"

	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/storage"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

func TestMarshal(t *testing.T) {
	t.Run("should returns an error if the type is not registered", func(t *testing.T) {
		_, err := bus.UnmarshalMessage(notRegistered{}.Name_(), "")

		testutil.ErrorIs(t, storage.ErrCouldNotUnmarshalGivenType, err)
	})

	t.Run("should be able to marshal a message", func(t *testing.T) {
		v, err := bus.MarshalMessage(addCommand{A: 1, B: 2})

		testutil.IsNil(t, err)
		testutil.Equals(t, `{"A":1,"B":2}`, v)
	})

	t.Run("should be able to unmarshal a registered type", func(t *testing.T) {
		bus.RegisterForMarshalling[addCommand]()

		r, err := bus.UnmarshalMessage(addCommand{}.Name_(), `{"A":1,"B":2}`)

		testutil.IsNil(t, err)
		testutil.Equals(t, addCommand{A: 1, B: 2}, r.(addCommand))
	})
}

type notRegistered struct {
	bus.Command[bus.UnitType]
}

func (notRegistered) Name_() string { return "NotRegistered" }
