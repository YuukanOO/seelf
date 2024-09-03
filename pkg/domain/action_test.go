package domain_test

import (
	"testing"
	"time"

	"github.com/YuukanOO/seelf/pkg/assert"
	"github.com/YuukanOO/seelf/pkg/domain"
	"github.com/YuukanOO/seelf/pkg/must"
)

type userId string

func Test_Action(t *testing.T) {
	t.Run("should creates a new action from a user", func(t *testing.T) {
		var user userId = "john"
		act := domain.NewAction(user)

		assert.Equal(t, user, act.By())
		assert.False(t, act.At().IsZero())
	})

	t.Run("should be rehydrated with the From function", func(t *testing.T) {
		var (
			user userId = "john"
			at          = must.Panic(time.Parse(time.RFC3339, "1989-03-09T08:00:00Z"))
		)

		act := domain.ActionFrom(user, at)

		assert.Equal(t, user, act.By())
		assert.Equal(t, at, act.At())
	})
}
