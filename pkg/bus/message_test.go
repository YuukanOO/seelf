package bus_test

import (
	"context"
	"testing"

	"github.com/YuukanOO/seelf/pkg/assert"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/storage"
)

func Test_Message(t *testing.T) {
	t.Run("Command should implements correct interfaces", func(t *testing.T) {
		t.Run("should implements bus.Message, bus.Request, bus.TypedRequest and bus.MutateRequest interfaces", func(t *testing.T) {
			var cmd addCommand

			_, ok := any(cmd).(bus.MutateRequest)
			assert.True(t, ok)

			_, ok = any(cmd).(bus.Request)
			assert.True(t, ok)

			_, ok = any(cmd).(bus.TypedRequest[int])
			assert.True(t, ok)

			_, ok = any(cmd).(bus.Message)
			assert.True(t, ok)
		})

		t.Run("should not implements bus.Signal, bus.AsyncRequest and bus.QueryRequest interface", func(t *testing.T) {
			var cmd addCommand

			_, ok := any(cmd).(bus.Signal)
			assert.False(t, ok)

			_, ok = any(cmd).(bus.QueryRequest)
			assert.False(t, ok)

			_, ok = any(cmd).(bus.AsyncRequest)
			assert.False(t, ok)
		})
	})

	t.Run("Query should implements correct interfaces", func(t *testing.T) {
		t.Run("should implements bus.Message, bus.Request, bus.TypedRequest and bus.QueryRequest interface", func(t *testing.T) {
			var cmd getQuery

			_, ok := any(cmd).(bus.Message)
			assert.True(t, ok)

			_, ok = any(cmd).(bus.Request)
			assert.True(t, ok)

			_, ok = any(cmd).(bus.TypedRequest[int])
			assert.True(t, ok)

			_, ok = any(cmd).(bus.QueryRequest)
			assert.True(t, ok)
		})

		t.Run("should not implements bus.Signal, bus.AsyncRequest and bus.MutateRequest interface", func(t *testing.T) {
			var cmd getQuery

			_, ok := any(cmd).(bus.Signal)
			assert.False(t, ok)

			_, ok = any(cmd).(bus.AsyncRequest)
			assert.False(t, ok)

			_, ok = any(cmd).(bus.MutateRequest)
			assert.False(t, ok)
		})
	})

	t.Run("AsyncCommand should implements correct interfaces", func(t *testing.T) {
		t.Run("should implements bus.Message, bus.Request, bus.TypedRequest, bus.MutateRequest and bus.AsyncRequest interface", func(t *testing.T) {
			var cmd asyncCommand

			_, ok := any(cmd).(bus.MutateRequest)
			assert.True(t, ok)

			_, ok = any(cmd).(bus.Request)
			assert.True(t, ok)

			_, ok = any(cmd).(bus.TypedRequest[bus.AsyncResult])
			assert.True(t, ok)

			_, ok = any(cmd).(bus.Message)
			assert.True(t, ok)

			_, ok = any(cmd).(bus.AsyncRequest)
			assert.True(t, ok)
		})

		t.Run("should not implements bus.Signal and bus.QueryRequest interface", func(t *testing.T) {
			var cmd asyncCommand

			_, ok := any(cmd).(bus.Signal)
			assert.False(t, ok)

			_, ok = any(cmd).(bus.QueryRequest)
			assert.False(t, ok)
		})
	})

	t.Run("Notification should implements correct interfaces", func(t *testing.T) {
		t.Run("should implements bus.Message and bus.Signal interface", func(t *testing.T) {
			var evt somethingCreated

			_, ok := any(evt).(bus.Message)
			assert.True(t, ok)

			_, ok = any(evt).(bus.Signal)
			assert.True(t, ok)
		})

		t.Run("should not implements bus.Request, bus.TypedRequest, bus.AsyncRequest, bus.MutateRequest and bus.QueryRequest interface", func(t *testing.T) {
			var evt somethingCreated

			_, ok := any(evt).(bus.Request)
			assert.False(t, ok)

			_, ok = any(evt).(bus.TypedRequest[int])
			assert.False(t, ok)

			_, ok = any(evt).(bus.AsyncRequest)
			assert.False(t, ok)

			_, ok = any(evt).(bus.MutateRequest)
			assert.False(t, ok)

			_, ok = any(evt).(bus.QueryRequest)
			assert.False(t, ok)
		})
	})

	t.Run("should be registerable on a bus", func(t *testing.T) {
		var b dummyBus

		bus.Register(b, func(context.Context, addCommand) (int, error) { return 0, nil })
		bus.Register(b, func(context.Context, getQuery) (int, error) { return 0, nil })
		bus.On(b, func(context.Context, somethingCreated) error { return nil })

		_, _ = bus.Send(b, context.Background(), addCommand{A: 1, B: 2})

		_, err := bus.Marshallable.From(addCommand{}.Name_(), "")
		assert.ErrorIs(t, storage.ErrCouldNotUnmarshalGivenType, err, "should not have been registered on the discriminated union mapper")
	})

	t.Run("should automatically register a mapper for async request", func(t *testing.T) {
		var b dummyBus

		bus.Register(b, func(context.Context, asyncCommand) (bus.AsyncResult, error) { return 0, nil })

		r, err := bus.Marshallable.From(asyncCommand{}.Name_(), `{"SomeValue":42}`)
		assert.Nil(t, err)
		assert.Equal(t, asyncCommand{
			SomeValue: 42,
		}, r.(asyncCommand))
	})
}

type dummyBus struct{}

func (dummyBus) Register(bus.Message, bus.NextFunc)             {}
func (dummyBus) Send(context.Context, bus.Request) (any, error) { return 0, nil }
func (dummyBus) Notify(context.Context, ...bus.Signal) error    { return nil }

type addCommand struct {
	bus.Command[int]

	A int
	B int
}

func (addCommand) Name_() string { return "AddCommand" }

type asyncCommand struct {
	bus.AsyncCommand

	SomeValue int
}

func (asyncCommand) Name_() string      { return "AsyncCommand" }
func (asyncCommand) ResourceID() string { return "" }
func (asyncCommand) Group() string      { return "" }

type getQuery struct {
	bus.Query[int]
}

func (getQuery) Name_() string { return "GetQuery" }

type somethingCreated struct {
	bus.Notification

	ID int
}

func (somethingCreated) Name_() string { return "SomethingCreated" }
