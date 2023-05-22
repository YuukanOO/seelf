package event

type (
	// Event triggered by main aggregates. This type is mostly a precaution in case
	// required metadata evolve afterwards.
	Event any

	// Represents an event source which contains events before dispatching.
	// Methods are unexported to avoid polluting the domain entities with those
	// considerations for external clients.
	//
	// You may use `Store` and `Unwrap` to manipulate a struct which embed an `Emitter`.
	Source interface {
		storeEvents(...Event)
		pendingEvents() []Event
		// clearEvents()
	}

	// Implements the Source interface, embed it in your own entities to enable
	// functions such as `Store` and `Unwrap`.
	//
	// I really like this solution because it does not poluate the domain stuff
	// by exposing methods on entities. You have to explicitly ask for it.
	Emitter struct {
		events []Event
	}
)

// Store given events in the given source.
// In order to use this method, you must embed the `event.Emitter` struct.
func Store(s Source, events ...Event) {
	s.storeEvents(events...)
}

// Unwrap events from the given source.
// In order to use this method, you must embed the `event.Emitter` struct.
func Unwrap(s Source) []Event {
	return s.pendingEvents()
}

// // Remove all events from the given source. This is needed to mark them has already
// // processed by the system.
// func Clear(s Source) {
// 	s.clearEvents()
// }

func (e *Emitter) storeEvents(events ...Event) {
	e.events = append(e.events, events...)
}

func (e *Emitter) pendingEvents() []Event {
	return e.events
}

// func (e *Emitter) clearEvents() {
// 	e.events = nil
// 	e.events = []Event{}
// }
