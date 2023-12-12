package bus

import (
	"context"
	"errors"
)

var ErrNoHandlerRegistered = errors.New("no_handler_registered")

type (
	// Handler for a specific message.
	RequestHandler[TResult any, TMsg Request[TResult]] func(context.Context, TMsg) (TResult, error)
	// Handler for signal.
	SignalHandler[TSignal Signal] func(context.Context, TSignal) error
	// Generic handler (as seen by middlewares).
	NextFunc func(context.Context, Message) (any, error)
	// Middleware function used to add behavior to the dispatch process.
	MiddlewareFunc func(NextFunc) NextFunc

	Bus interface {
		register(Message, NextFunc) // Register an handler for a specific message kind
		handler(string) any         // Get the handler for a specific message kind (any since for notification it will be an array of handlers)
	}

	inMemoryBus struct {
		middlewares []MiddlewareFunc
		handlers    map[string]any
	}
)

// Register an handler for a specific request on the provided bus.
func Register[TResult any, TMsg Request[TResult]](bus Bus, handler RequestHandler[TResult, TMsg]) {
	var (
		msg TMsg
		h   NextFunc = func(ctx context.Context, m Message) (any, error) {
			return handler(ctx, m.(TMsg))
		}
	)

	bus.register(msg, h)
}

// Register a signal handler for the given signal. Multiple signals can be registered for the same signal
// and will all be called.
func On[TSignal Signal](bus Bus, handler SignalHandler[TSignal]) {
	var (
		msg TSignal
		h   NextFunc = func(ctx context.Context, m Message) (any, error) {
			return nil, handler(ctx, m.(TSignal))
		}
	)

	bus.register(msg, h)
}

// Send the given message to the bus and return the result and an error if any.
func Send[TResult any, TMsg Request[TResult]](bus Bus, ctx context.Context, msg TMsg) (TResult, error) {
	handler := bus.handler(msg.Name_())

	if handler == nil {
		var r TResult
		return r, ErrNoHandlerRegistered
	}

	r, err := handler.(NextFunc)(ctx, msg)

	return r.(TResult), err
}

// Call every signal handlers registered for given signals.
func Notify(bus Bus, ctx context.Context, msgs ...Signal) error {
	for _, msg := range msgs {
		handlers := bus.handler(msg.Name_())

		if handlers == nil {
			return nil
		}

		hdls := handlers.([]NextFunc)

		for _, h := range hdls {
			_, err := h(ctx, msg)

			if err != nil {
				return err
			}
		}
	}

	return nil
}

// Creates a new in memory bus which will call the handlers synchronously.
func NewInMemoryBus(middlewares ...MiddlewareFunc) Bus {
	return &inMemoryBus{
		middlewares: middlewares,
		handlers:    make(map[string]any),
	}
}

func (b *inMemoryBus) register(msg Message, handler NextFunc) {
	name := msg.Name_()
	_, exists := b.handlers[name]

	// Apply middlewares to avoid doing it at runtime
	for i := len(b.middlewares) - 1; i >= 0; i-- {
		handler = b.middlewares[i](handler)
	}

	if msg.Kind_() == MessageKindNotification {
		if !exists {
			b.handlers[name] = []NextFunc{handler}
		} else {
			b.handlers[name] = append(b.handlers[name].([]NextFunc), handler)
		}
		return
	}

	if exists {
		panic("an handler is already registered for " + name) // Panic since this should never happen outside of a dev environment
	}

	b.handlers[name] = handler
}

func (b *inMemoryBus) handler(name string) any {
	return b.handlers[name]
}
