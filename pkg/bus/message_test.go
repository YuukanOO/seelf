package bus_test

import (
	"testing"

	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

func TestMessage(t *testing.T) {
	t.Run("should have the appropriate kind", func(t *testing.T) {
		var (
			command AddCommand
			query   GetQuery
			notif   RegisteredNotification
		)

		testutil.Equals(t, bus.MessageKindCommand, command.Kind_())
		testutil.Equals(t, bus.MessageKindQuery, query.Kind_())
		testutil.Equals(t, bus.MessageKindNotification, notif.Kind_())
	})
}

type AddCommand struct {
	bus.Command[int]

	A int
	B int
}

func (AddCommand) Name_() string { return "AddCommand" }

type GetQuery struct {
	bus.Query[int]
}

func (GetQuery) Name_() string { return "GetQuery" }

type RegisteredNotification struct {
	bus.Notification

	Id int
}

func (RegisteredNotification) Name_() string { return "RegisteredNotification" }
