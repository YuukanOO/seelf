package event_test

import (
	"testing"

	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/event"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

type (
	domainEntity struct {
		event.Emitter

		name string
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

		testutil.HasLength(t, evts, 2)
		testutil.Equals(t, evt1, evts[0].(domainEventA))
		testutil.Equals(t, evt2, evts[1].(domainEventB))
	})

	// t.Run("should be able to clear all events from an Emitter", func(t *testing.T) {
	// 	ent := domainEntity{}
	// 	Store(&ent, domainEventA{}, domainEventB{})

	// 	Clear(&ent)

	// 	testutil.HasLength(t, Unwrap(&ent), 0)
	// })

	t.Run("should mutate the right array in a pure entity", func(t *testing.T) {
		ent := domainEntity{}
		evt1 := domainEventA{}
		evt2 := domainEventB{}

		ent1 := ent.apply(evt1)
		ent2 := ent1.apply(evt2)

		events := event.Unwrap(&ent)
		events1 := event.Unwrap(&ent1)
		events2 := event.Unwrap(&ent2)

		testutil.Equals(t, "", ent.name)
		testutil.HasLength(t, events, 0)
		testutil.Equals(t, "eventA", ent1.name)
		testutil.HasLength(t, events1, 1)
		testutil.Equals(t, evt1, events1[0].(domainEventA))

		testutil.Equals(t, "eventB", ent2.name)
		testutil.HasLength(t, events2, 2)
		testutil.Equals(t, evt1, events2[0].(domainEventA))
		testutil.Equals(t, evt2, events2[1].(domainEventB))
	})
}

func (d domainEntity) apply(e event.Event) domainEntity {
	switch e.(type) {
	case domainEventA:
		d.name = "eventA"
	case domainEventB:
		d.name = "eventB"
	}

	event.Store(&d, e)
	return d
}
