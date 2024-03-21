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

		data string
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

	t.Run("should replace an event if it already exists", func(t *testing.T) {
		ent := domainEntity{}

		ent.apply(domainEventA{})
		ent.apply(domainEventB{})
		ent.apply(domainEventA{})

		event.Replace(&ent, domainEventB{data: "updated one"})

		testutil.HasNEvents(t, &ent, 3)
		evt := testutil.EventIs[domainEventB](t, &ent, 2)
		testutil.Equals(t, "updated one", evt.data)
	})
}

func (d *domainEntity) apply(e event.Event) {
	switch e.(type) {
	case domainEventA:
		d.name = "eventA"
	case domainEventB:
		d.name = "eventB"
	}

	event.Store(d, e)
}
