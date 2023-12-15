// Mediator style message bus adapted to the Go language without requiring the reflect package.
// Message members are suffixed with an underscore to avoid name conflicts it a command need a field
// named "Name" for example.
//
// Performance wise, it is lightly slower than direct calls but you get a lower coupling
// and a way to add middlewares to your handlers.
package bus

import "github.com/YuukanOO/seelf/pkg/storage"

const (
	MessageKindNotification MessageKind = iota
	MessageKindCommand
	MessageKindQuery
)

// Constant unit value to return when a request does not need a specific result set.
// In my mind, it should avoid the cost of allocating something not needed.
const Unit UnitType = iota

// Contains message which can be unmarshalled from a raw string (= those used in the scheduler).
var Marshallable = storage.NewDiscriminatedMapper(func(r Request) string { return r.Name_() })

type (
	// Sometimes, you may not need a result type on a request but the RequestHandler expect one, just
	// use this type as the result type and the bus.Unit as the return value.
	UnitType uint8

	// Represent the kind of a message being dispatched. This is especially useful for middlewares
	// to adapt their behavior depending on the message kind.
	//
	// For example, a command may need a transaction whereas a query may not.
	MessageKind uint8

	// Message which can be sent in the bus and handled by a registered handler.
	Message interface {
		Name_() string      // Unique name of the message (here to not require reflection)
		Kind_() MessageKind // Type of the message to be able to customize middlewares
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

	// Message without result implementing the Signal interface.
	Notification struct{}

	// Request to mutate the system. Implements the TypedRequest interface.
	Command[T any] struct{}

	// Request to query the system. Implements the TypedRequest interface.
	Query[T any] struct{}
)

func (Notification) Kind_() MessageKind { return MessageKindNotification }
func (Notification) isSignal()          {}

func (Command[T]) Kind_() MessageKind    { return MessageKindCommand }
func (Command[T]) isRequest()            {}
func (Command[T]) isTypedRequest() (t T) { return t }

func (Query[T]) Kind_() MessageKind    { return MessageKindQuery }
func (Query[T]) isRequest()            {}
func (Query[T]) isTypedRequest() (t T) { return t }
