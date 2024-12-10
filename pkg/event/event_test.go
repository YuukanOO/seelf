package event_test

import (
	"testing"
	"time"

	"github.com/YuukanOO/seelf/pkg/assert"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/event"
)

type (
	domainEntity struct {
		event.Emitter
	}

	domainEventA struct {
		bus.Notification
	}

	domainEventB struct {
		bus.Notification
	}
)

func (domainEventA) Name_() string { return "domain_event_a" }
func (domainEventB) Name_() string { return "domain_event_b" }

func Test_Emitter(t *testing.T) {
	t.Run("version should be zero by default", func(t *testing.T) {
		ent := domainEntity{}

		version, events := event.Unwrap(&ent)

		assert.Zero(t, version)
		assert.HasLength(t, 0, events)
	})

	t.Run("could be hydrated and remove events when doing so", func(t *testing.T) {
		ent := domainEntity{}
		event.Store(&ent, domainEventA{}, domainEventB{})
		newVersion := time.Now().UTC()
		event.Hydrate(&ent, newVersion)

		version, events := event.Unwrap(&ent)

		assert.Equal(t, newVersion, version)
		assert.HasLength(t, 0, events)
	})

	t.Run("should be able to store and retrieve events from an Emitter", func(t *testing.T) {
		ent := domainEntity{}
		evt1 := domainEventA{}
		evt2 := domainEventB{}
		event.Store(&ent, evt1, evt2)

		version, events := event.Unwrap(&ent)

		assert.HasLength(t, 2, events)
		assert.Zero(t, version)
		assert.Equal(t, evt1, events[0].(domainEventA))
		assert.Equal(t, evt2, events[1].(domainEventB))
	})
}
