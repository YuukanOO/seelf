package bus_test

import (
	"context"
	"errors"
	"testing"

	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

func TestInMemoryBus(t *testing.T) {
	t.Run("should accepts registration of all message kind", func(t *testing.T) {
		local := bus.NewInMemoryBus()

		bus.Register(local, AddCommandHandler)
		bus.Register(local, GetQueryHandler)
		bus.On(local, NotificationHandler)
		bus.On(local, OtherNotificationHandler)
	})

	t.Run("should panic if an handler is already registered for a request", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("should have panicked")
			}
		}()

		local := bus.NewInMemoryBus()

		bus.Register(local, AddCommandHandler)
		bus.Register(local, AddCommandHandler)
	})

	t.Run("should returns an error if no handler is registered for a given request", func(t *testing.T) {
		local := bus.NewInMemoryBus()

		_, err := bus.Send(local, context.Background(), &AddCommand{})

		testutil.ErrorIs(t, bus.ErrNoHandlerRegistered, err)
	})

	t.Run("should returns the request handler error back if any", func(t *testing.T) {
		local := bus.NewInMemoryBus()
		expectedErr := errors.New("handler error")

		bus.Register(local, func(ctx context.Context, cmd AddCommand) (int, error) {
			return 0, expectedErr
		})

		_, err := bus.Send(local, context.Background(), AddCommand{})

		testutil.ErrorIs(t, expectedErr, err)
	})

	t.Run("should call the appropriate request handler and returns the result", func(t *testing.T) {
		local := bus.NewInMemoryBus()

		bus.Register(local, AddCommandHandler)
		bus.Register(local, GetQueryHandler)
		bus.On(local, NotificationHandler)
		bus.On(local, OtherNotificationHandler)

		result, err := bus.Send(local, context.Background(), AddCommand{A: 1, B: 2})

		testutil.IsNil(t, err)
		testutil.Equals(t, 3, result)

		result, err = bus.Send(local, context.Background(), GetQuery{})

		testutil.IsNil(t, err)
		testutil.Equals(t, 42, result)
	})

	t.Run("should do nothing if no signal handler is registered for a given signal", func(t *testing.T) {
		local := bus.NewInMemoryBus()

		err := bus.Notify(local, context.Background(), RegisteredNotification{})

		testutil.IsNil(t, err)
	})

	t.Run("should returns a signal handler error back if any", func(t *testing.T) {
		local := bus.NewInMemoryBus()
		expectedErr := errors.New("handler error")

		bus.On(local, func(ctx context.Context, notif RegisteredNotification) error {
			return nil
		})

		bus.On(local, func(ctx context.Context, notif RegisteredNotification) error {
			return expectedErr
		})

		err := bus.Notify(local, context.Background(), RegisteredNotification{})

		testutil.ErrorIs(t, expectedErr, err)
	})

	t.Run("should call every signal handlers registered for the given signal", func(t *testing.T) {
		var (
			local           = bus.NewInMemoryBus()
			firstOneCalled  = false
			secondOneCalled = false
		)

		bus.On(local, func(ctx context.Context, notif RegisteredNotification) error {
			firstOneCalled = true
			return nil
		})

		bus.On(local, func(ctx context.Context, notif RegisteredNotification) error {
			secondOneCalled = true
			return nil
		})

		err := bus.Notify(local, context.Background(), RegisteredNotification{})

		testutil.IsNil(t, err)
		testutil.IsTrue(t, firstOneCalled && secondOneCalled)
	})

	t.Run("should call every middlewares registered", func(t *testing.T) {
		calls := make([]int, 0)

		local := bus.NewInMemoryBus(
			func(next bus.NextFunc) bus.NextFunc {
				return func(ctx context.Context, m bus.Message) (any, error) {
					calls = append(calls, 1)
					r, err := next(ctx, m)
					calls = append(calls, 1)
					return r, err
				}
			},
			func(next bus.NextFunc) bus.NextFunc {
				return func(ctx context.Context, m bus.Message) (any, error) {
					calls = append(calls, 2)
					r, err := next(ctx, m)
					calls = append(calls, 2)
					return r, err
				}
			},
		)

		bus.Register(local, AddCommandHandler)
		bus.Register(local, GetQueryHandler)
		bus.On(local, NotificationHandler)
		bus.On(local, OtherNotificationHandler)

		r, err := bus.Send(local, context.Background(), AddCommand{
			A: 1,
			B: 2,
		})

		testutil.IsNil(t, err)
		testutil.Equals(t, 3, r)
		testutil.DeepEquals(t, []int{1, 2, 2, 1}, calls)

		calls = make([]int, 0)

		bus.Notify(local, context.Background(), RegisteredNotification{})

		// Should have been called twice cuz 2 signal handlers are registered
		testutil.DeepEquals(t, []int{1, 2, 2, 1, 1, 2, 2, 1}, calls)
	})
}

func AddCommandHandler(ctx context.Context, cmd AddCommand) (int, error) {
	return cmd.A + cmd.B, nil
}

func GetQueryHandler(ctx context.Context, query GetQuery) (int, error) {
	return 42, nil
}

func NotificationHandler(ctx context.Context, notif RegisteredNotification) error {
	return nil
}

func OtherNotificationHandler(ctx context.Context, notif RegisteredNotification) error {
	return nil
}
