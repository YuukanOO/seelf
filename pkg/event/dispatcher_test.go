package event_test

import (
	"context"
	"errors"
	"testing"

	"github.com/YuukanOO/seelf/pkg/event"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

func Test_InProcessDispatcher(t *testing.T) {
	t.Run("should call each handler when an event is dispatched", func(t *testing.T) {
		var (
			evt1 = domainEventA{}
			evt2 = domainEventB{}

			called1 []event.Event
			called2 []event.Event

			handler1 event.Handler = func(ctx context.Context, e event.Event) error {
				called1 = append(called1, e)
				return nil
			}

			handler2 event.Handler = func(ctx context.Context, e event.Event) error {
				called2 = append(called2, e)
				return nil
			}
		)

		dispatcher := event.NewInProcessDispatcher(handler1, handler2)
		err := dispatcher.Dispatch(context.Background(), evt1, evt2)

		testutil.IsNil(t, err)
		testutil.HasLength(t, called1, 2)
		testutil.HasLength(t, called2, 2)
		testutil.Equals(t, evt1, called1[0].(domainEventA))
		testutil.Equals(t, evt2, called1[1].(domainEventB))
		testutil.Equals(t, evt1, called2[0].(domainEventA))
		testutil.Equals(t, evt2, called2[1].(domainEventB))
	})

	t.Run("should returns early if an error has occurred", func(t *testing.T) {
		var (
			errReturned = errors.New("an unexpected error")
			evt1        = domainEventA{}
			evt2        = domainEventB{}

			called2 []event.Event

			handler1 event.Handler = func(ctx context.Context, e event.Event) error {
				return errReturned
			}

			handler2 event.Handler = func(ctx context.Context, e event.Event) error {
				called2 = append(called2, e)
				return nil
			}
		)

		dispatcher := event.NewInProcessDispatcher(handler1, handler2)
		err := dispatcher.Dispatch(context.Background(), evt1, evt2)

		testutil.ErrorIs(t, errReturned, err)
		testutil.HasLength(t, called2, 0)
	})
}
