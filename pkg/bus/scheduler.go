package bus

import (
	"context"
)

// Enable scheduled dispatching of a message.
type Scheduler interface {
	// Queue a request to be dispatched asynchronously at a later time.
	Queue(context.Context, ...AsyncRequest) error
	// Cancel scheduled jobs represented by the given keys.
	Cancel(context.Context, ...JobKey) error
}
