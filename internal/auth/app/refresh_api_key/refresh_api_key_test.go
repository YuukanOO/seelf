package refresh_api_key_test

import (
	"context"
	"testing"

	"github.com/YuukanOO/seelf/internal/auth/app/refresh_api_key"
	"github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/internal/auth/infra/crypto"
	"github.com/YuukanOO/seelf/internal/auth/infra/memory"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/must"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

func Test_RefreshApiKey(t *testing.T) {
	sut := func(existingUsers ...*domain.User) bus.RequestHandler[string, refresh_api_key.Command] {
		store := memory.NewUsersStore(existingUsers...)

		return refresh_api_key.Handler(store, store, crypto.NewKeyGenerator())
	}

	t.Run("should fail if the user does not exists", func(t *testing.T) {
		uc := sut()

		_, err := uc(context.Background(), refresh_api_key.Command{})

		testutil.ErrorIs(t, apperr.ErrNotFound, err)
	})

	t.Run("should refresh the user's API key if everything is good", func(t *testing.T) {
		user := must.Panic(domain.NewUser(domain.NewEmailRequirement("some@email.com", true), "someHashedPassword", "apikey"))
		uc := sut(&user)

		key, err := uc(context.Background(), refresh_api_key.Command{
			ID: string(user.ID())},
		)

		testutil.IsNil(t, err)
		testutil.NotEquals(t, "", key)

		evt := testutil.EventIs[domain.UserAPIKeyChanged](t, &user, 1)

		testutil.Equals(t, user.ID(), evt.ID)
		testutil.Equals(t, key, string(evt.Key))
	})
}
