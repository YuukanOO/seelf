package bus

import (
	"context"
	"errors"
)

var ErrNoHandlerRegistered = errors.New("no_handler_registered")

type (
	// Handler for a specific message.
	RequestHandler[TResult any, TMsg TypedRequest[TResult]] func(context.Context, TMsg) (TResult, error)
	// Handler for signal.
	SignalHandler[TSignal Signal] func(context.Context, TSignal) error
	// Generic handler (as seen by middlewares).
	NextFunc func(context.Context, Message) (any, error)

	// Dispatcher is the interface used to send messages to the bus.
	Dispatcher interface {
		Send(ctx context.Context, msg Request) (any, error) // Send the given request message to the bus
		Notify(ctx context.Context, msgs ...Signal) error   // Call every signal handlers attached to the given signals
	}

	// Dispatcher with registration capabilities.
	Bus interface {
		Dispatcher
		Register(Message, NextFunc) // Register an handler for a specific message kind, even if you can use this method directly, you should prefer the typed version bus.Register and bus.On
	}
)

// Register an handler for a specific request on the provided bus.
func Register[TResult any, TMsg TypedRequest[TResult]](bus Bus, handler RequestHandler[TResult, TMsg]) {
	var (
		msg TMsg
		h   NextFunc = func(ctx context.Context, m Message) (any, error) {
			return handler(ctx, m.(TMsg))
		}
	)

	bus.Register(msg, h)
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

	bus.Register(msg, h)
}

// Send the given message to the bus and return the result. This method ensure type safety when dispatching
// a request.
func Send[TResult any, TMsg TypedRequest[TResult]](bus Dispatcher, ctx context.Context, msg TMsg) (TResult, error) {
	r, err := bus.Send(ctx, msg)

	if errors.Is(err, ErrNoHandlerRegistered) {
		var tr TResult
		return tr, err
	}

	return r.(TResult), err
}
