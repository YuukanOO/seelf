package domain_test

import (
	"context"
	"testing"

	"github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

func Test_Auth_Context(t *testing.T) {
	t.Run("should embed a user id into the context", func(t *testing.T) {
		ctx := context.Background()
		uid := domain.UserID("auserid")

		newCtx := domain.WithUserID(ctx, uid)

		testutil.Equals(t, uid, domain.CurrentUser(newCtx).MustGet())
	})

	t.Run("should returns an empty monad.Maybe if no user id has been attached to the context", func(t *testing.T) {
		uid := domain.CurrentUser(context.Background())

		testutil.IsFalse(t, uid.HasValue())
	})
}
