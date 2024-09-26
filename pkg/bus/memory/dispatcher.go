package memory

import (
	"context"

	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/types"
)

type (
	// Middleware function used to add behavior to the dispatch process.
	MiddlewareFunc func(bus.NextFunc) bus.NextFunc

	dispatcher struct {
		middlewares []MiddlewareFunc
		handlers    map[string]any
	}
)

// Creates a new in memory bus which will call the handlers in process.
func NewBus(middlewares ...MiddlewareFunc) bus.Bus {
	return &dispatcher{
		middlewares: middlewares,
		handlers:    make(map[string]any),
	}
}

func (b *dispatcher) Register(msg bus.Message, handler bus.NextFunc) {
	name := msg.Name_()
	_, exists := b.handlers[name]

	// Apply middlewares here to avoid doing it at runtime
	for i := len(b.middlewares) - 1; i >= 0; i-- {
		handler = b.middlewares[i](handler)
	}

	if types.Is[bus.Signal](msg) {
		if !exists {
			b.handlers[name] = []bus.NextFunc{handler}
		} else {
			b.handlers[name] = append(b.handlers[name].([]bus.NextFunc), handler)
		}
		return
	}

	if exists {
		panic("an handler is already registered for " + name) // Panic since this should never happen outside of a dev environment
	}

	b.handlers[name] = handler
}

func (b *dispatcher) Send(ctx context.Context, msg bus.Request) (any, error) {
	handler := b.handlers[msg.Name_()]

	if handler == nil {
		return nil, bus.ErrNoHandlerRegistered
	}

	return handler.(bus.NextFunc)(ctx, msg)
}

func (b *dispatcher) Notify(ctx context.Context, msgs ...bus.Signal) error {
	for _, msg := range msgs {
		value := b.handlers[msg.Name_()]

		if value == nil {
			continue
		}

		handlers := value.([]bus.NextFunc)

		for _, h := range handlers {
			_, err := h(ctx, msg)

			if err != nil {
				return err
			}
		}
	}

	return nil
}
