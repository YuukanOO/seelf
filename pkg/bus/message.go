// Mediator style message bus adapted to the Go language without requiring the reflect package.
// Message members are suffixed with an underscore to avoid name conflicts it a command need a field
// named "Name" for example.
//
// Performance wise, it is lightly slower than direct calls but you get a lower coupling
// and a way to add middlewares to your handlers.
package bus

import (
	"github.com/YuukanOO/seelf/pkg/storage"
)

// Constant unit value to return when a request does not need a specific result set.
// In my mind, it should avoid the cost of allocating something not needed.
const Unit UnitType = iota

const (
	AsyncResultProcessed AsyncResult = iota // Async job has been handled correctly (could have failed but in this case the error will not be nil)
	AsyncResultDelay                        // Async job could not be processed right now and should be retried later
)

// Contains message which can be unmarshalled from a raw string (= those used in the scheduler).
var Marshallable = storage.NewDiscriminatedMapper(func(r AsyncRequest) string { return r.Name_() })

type (
	// Sometimes, you may not need a result type on a request but the RequestHandler expect one, just
	// use this type as the result type and the bus.Unit as the return value.
	UnitType uint8

	// Async result of an async request, one processed by the scheduler.
	AsyncResult uint8

	// Message which can be sent in the bus and handled by a registered handler.
	Message interface {
		Name_() string // Unique name of the message (here to not require reflection)
	}

	// Signal which do not need a result.
	Signal interface {
		Message
		isSignal() // Marker method to differentiate signals from requests
	}

	// Message which requires a result.
	Request interface {
		Message
		isRequest() // Marker method to differentiate requests from signals
	}

	// Request with a typed result.
	TypedRequest[T any] interface {
		Request
		isTypedRequest() T // Marker method. Without it, the compiler will not be able to infer the T.
	}

	// Request to mutate the system.
	MutateRequest interface {
		Request
		isMutateRequest()
	}

	// Async request which extend the MutateRequest with additional information.
	// When implementing an AsyncRequest handler, you must make it idempotent since a job
	// can be retried under certain conditions.
	AsyncRequest interface {
		MutateRequest
		TypedRequest[AsyncResult]
		ResourceID() string // ID of the main resource processed by the request
		Group() string      // Work group for this request, at most one job per group is processed at any given time
	}

	// Request to query the system.
	QueryRequest interface {
		Request
		isQueryRequest()
	}

	// Message without result implementing the Signal interface.
	Notification struct{}

	// Request to mutate the system. Implements the TypedRequest and MutateRequest interface.
	// The Name_() method is not implemented by this to make sure you do not forget to declare
	// it.
	Command[T any] struct{}

	// Async command for stuff that should be processed in the background by the Scheduler.
	AsyncCommand struct {
		Command[AsyncResult]
	}

	// Request to query the system. Implements the TypedRequest and QueryRequest interface.
	// The Name_() method is not implemented by this to make sure you do not forget to declare
	// it.
	Query[T any] struct{}
)

func (Notification) isSignal() {}

func (Command[T]) isRequest()            {}
func (Command[T]) isMutateRequest()      {}
func (Command[T]) isTypedRequest() (t T) { return t }

func (Query[T]) isRequest()            {}
func (Query[T]) isQueryRequest()       {}
func (Query[T]) isTypedRequest() (t T) { return t }
