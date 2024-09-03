package bus_test

import (
	"testing"

	"github.com/YuukanOO/seelf/pkg/assert"
	"github.com/YuukanOO/seelf/pkg/bus"
)

func TestMessage(t *testing.T) {
	t.Run("should have the appropriate kind", func(t *testing.T) {
		var (
			command addCommand
			query   getQuery
			notif   registeredNotification
		)

		assert.Equal(t, bus.MessageKindCommand, command.Kind_())
		assert.Equal(t, bus.MessageKindQuery, query.Kind_())
		assert.Equal(t, bus.MessageKindNotification, notif.Kind_())
	})
}

type addCommand struct {
	bus.Command[int]

	A int
	B int
}

func (addCommand) Name_() string      { return "AddCommand" }
func (addCommand) ResourceID() string { return "" }

type getQuery struct {
	bus.Query[int]
}

func (getQuery) Name_() string { return "GetQuery" }

type registeredNotification struct {
	bus.Notification

	Id int
}

func (registeredNotification) Name_() string { return "RegisteredNotification" }
