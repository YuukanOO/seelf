package spy_test

import (
	"context"
	"testing"

	"github.com/YuukanOO/seelf/pkg/assert"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/bus/spy"
)

func Test_Dispatcher(t *testing.T) {
	t.Run("should store dispatched requests", func(t *testing.T) {
		dispatcher := spy.NewDispatcher()

		var req request

		_, err := dispatcher.Send(context.Background(), req)

		assert.Nil(t, err)
		assert.HasLength(t, 1, dispatcher.Requests())
		assert.Equal(t, req, dispatcher.Requests()[0].(request))
	})

	t.Run("should store dispatched signals", func(t *testing.T) {
		dispatcher := spy.NewDispatcher()

		var sig signal

		err := dispatcher.Notify(context.Background(), sig)

		assert.Nil(t, err)
		assert.HasLength(t, 1, dispatcher.Signals())
		assert.Equal(t, sig, dispatcher.Signals()[0].(signal))
	})

	t.Run("could reset dispatched requests and signals", func(t *testing.T) {
		dispatcher := spy.NewDispatcher()

		_, _ = dispatcher.Send(context.Background(), request{})
		_ = dispatcher.Notify(context.Background(), signal{})

		dispatcher.Reset()

		assert.HasLength(t, 0, dispatcher.Requests())
		assert.HasLength(t, 0, dispatcher.Signals())
	})
}

type request struct {
	bus.Command[int]
}

func (request) Name_() string { return "request" }

type signal struct {
	bus.Signal
}
