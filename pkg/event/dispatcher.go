package event

import "context"

type (
	// Represents an element capable of dispatching domain events somewhere.
	Dispatcher interface {
		Dispatch(context.Context, ...Event) error // Dispatch given events to appropriate handlers.
	}

	inProcessDispatcher struct {
		handlers []Handler
	}

	Handler func(context.Context, Event) error
)

// Instantiates a dispatcher which will dispatch events synchronously to the given
// handlers and returns early if an error has been thrown by one handler.
//
// This is the simplest dispatcher I can think of. It is mostly used to trigger
// usecases on some domain events to make them easier to test and reason about
// in isolation.
func NewInProcessDispatcher(handlers ...Handler) Dispatcher {
	return &inProcessDispatcher{
		handlers: handlers,
	}
}

func (d *inProcessDispatcher) Dispatch(ctx context.Context, events ...Event) error {
	for _, evt := range events {
		for _, handler := range d.handlers {
			if err := handler(ctx, evt); err != nil {
				return err
			}
		}
	}

	return nil
}
