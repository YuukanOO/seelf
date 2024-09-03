package domain_test

import (
	"context"
	"testing"

	"github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/pkg/assert"
	"github.com/YuukanOO/seelf/pkg/monad"
)

func Test_AuthContext(t *testing.T) {
	t.Run("should embed a user id into the context", func(t *testing.T) {
		ctx := context.Background()
		uid := domain.UserID("a_user_id")

		newCtx := domain.WithUserID(ctx, uid)

		assert.Equal(t, uid, domain.CurrentUser(newCtx).MustGet())
	})

	t.Run("should returns an empty monad.Maybe if no user id has been attached to the context", func(t *testing.T) {
		uid := domain.CurrentUser(context.Background())

		assert.Equal(t, monad.None[domain.UserID](), uid)
	})
}
