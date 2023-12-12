// Mediator style message bus adapted to the Go language without requiring the reflect package.
// Message members are suffixed with an underscore to avoid name conflicts it a command need a field
// named "Name" for example.
//
// Performance wise, it is lightly slower than direct calls but you get a lower coupling
// and a way to add middlewares to your handlers.
package bus

const (
	MessageKindNotification MessageKind = iota
	MessageKindCommand
	MessageKindQuery
)

type (
	// Represent the kind of a message being dispatched. This is especially useful for middlewares
	// to adapt their behavior depending on the message kind.
	//
	// For example, a command may need a transaction whereas a query may not.
	MessageKind int8

	// Message which can be sent in the bus and handled by a registered handler.
	Message interface {
		Name_() string      // Unique name of the message (here to not require reflection)
		Kind_() MessageKind // Type of the message to be able to customize middlewares
	}

	// Signal which do not need a result.
	Signal interface {
		Message
		isSignal() // Marker method to differentiate signals from messages
	}

	// Message which requires a result.
	Request[T any] interface {
		Message
		isRequest() T // Marker method. Without it, the compiler will not be able to infer the T.
	}

	// Message without result implementing the Signal interface.
	Notification struct{}

	// Request to mutate the system.
	Command[T any] struct{}

	// Request to query the system.
	Query[T any] struct{}
)

func (Notification) Kind_() MessageKind { return MessageKindNotification }
func (Notification) isSignal()          {}

func (Command[T]) Kind_() MessageKind { return MessageKindCommand }
func (Command[T]) isRequest() (t T)   { return t }

func (Query[T]) Kind_() MessageKind { return MessageKindQuery }
func (Query[T]) isRequest() (t T)   { return t }
