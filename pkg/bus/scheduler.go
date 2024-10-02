package bus

import "context"

// Enable scheduled dispatching of a message.
type Scheduler interface {
	// Queue a request to be dispatched asynchronously at a later time.
	Queue(context.Context, AsyncRequest) error
}
