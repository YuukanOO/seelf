package domain_test

import (
	"testing"
	"time"

	"github.com/YuukanOO/seelf/pkg/domain"
	"github.com/YuukanOO/seelf/pkg/must"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

type userId string

func Test_Action(t *testing.T) {
	t.Run("should creates a new action from a user", func(t *testing.T) {
		var user userId = "john"
		act := domain.NewAction(user)

		testutil.Equals(t, user, act.By())
		testutil.IsFalse(t, act.At().IsZero())
	})

	t.Run("should be rehydrated with the From function", func(t *testing.T) {
		var (
			user userId = "john"
			at          = must.Panic(time.Parse(time.RFC3339, "1989-03-09T08:00:00Z"))
		)

		act := domain.ActionFrom(user, at)

		testutil.Equals(t, user, act.By())
		testutil.Equals(t, at, act.At())
	})
}
