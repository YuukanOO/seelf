package bus_test

import (
	"testing"

	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

func TestMessage(t *testing.T) {
	t.Run("should have the appropriate kind", func(t *testing.T) {
		var (
			command addCommand
			query   getQuery
			notif   registeredNotification
		)

		testutil.Equals(t, bus.MessageKindCommand, command.Kind_())
		testutil.Equals(t, bus.MessageKindQuery, query.Kind_())
		testutil.Equals(t, bus.MessageKindNotification, notif.Kind_())
	})
}

type addCommand struct {
	bus.Command[int]

	A int
	B int
}

func (addCommand) Name_() string { return "AddCommand" }

type getQuery struct {
	bus.Query[int]
}

func (getQuery) Name_() string { return "GetQuery" }

type registeredNotification struct {
	bus.Notification

	Id int
}

func (registeredNotification) Name_() string { return "RegisteredNotification" }
