package event

import (
	"time"

	"github.com/YuukanOO/seelf/pkg/bus"
)

type (
	// Event triggered by main aggregates.
	Event bus.Signal

	// Simple alias (to make it easier to scan) for expressiveness
	Version = time.Time

	// Represents an event source which contains events before dispatching.
	// Methods are unexported to avoid polluting the domain entities with those
	// considerations for external clients.
	//
	// You may use `Store` and `Unwrap` to manipulate a struct which embed an `Emitter`.
	Source interface {
		storeEvents(...Event)
		pendingEvents() (Version, []Event)
		hydrateVersion(Version)
	}

	// Implements the Source interface, embed it in your own entities to enable
	// functions such as `Store` and `Unwrap`.
	//
	// I really like this solution because it does not pollute the domain stuff
	// by exposing methods on entities. You have to explicitly ask for it.
	Emitter struct {
		version Version
		events  []Event
	}
)

// Hydrate the given source with the given version, deleting any pending events in the
// the process.
func Hydrate(s Source, version Version) {
	s.hydrateVersion(version)
}

// Store given events in the given source.
// In order to use this method, you must embed the `event.Emitter` struct.
func Store(s Source, events ...Event) {
	s.storeEvents(events...)
}

// Unwrap events from the given source and returns them with the current version.
// In order to use this method, you must embed the `event.Emitter` struct.
func Unwrap(s Source) (Version, []Event) {
	return s.pendingEvents()
}

func (e *Emitter) storeEvents(events ...Event) {
	e.events = append(e.events, events...)
}

func (e *Emitter) pendingEvents() (Version, []Event) {
	return e.version, e.events
}

func (e *Emitter) hydrateVersion(t Version) {
	e.events = nil
	e.version = t
}
