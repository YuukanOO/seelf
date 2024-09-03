package event_test

import (
	"testing"

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
	t.Run("should be able to store and retrieve events from an Emitter", func(t *testing.T) {
		ent := domainEntity{}
		evt1 := domainEventA{}
		evt2 := domainEventB{}
		event.Store(&ent, evt1, evt2)

		evts := event.Unwrap(&ent)

		assert.HasLength(t, 2, evts)
		assert.Equal(t, evt1, evts[0].(domainEventA))
		assert.Equal(t, evt2, evts[1].(domainEventB))
	})

	// t.Run("should be able to clear all events from an Emitter", func(t *testing.T) {
	// 	ent := domainEntity{}
	// 	Store(&ent, domainEventA{}, domainEventB{})

	// 	Clear(&ent)

	// 	testutil.HasLength(t, Unwrap(&ent), 0)
	// })
}
