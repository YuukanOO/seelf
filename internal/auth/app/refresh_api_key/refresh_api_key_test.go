package refresh_api_key_test

import (
	"context"
	"testing"

	"github.com/YuukanOO/seelf/internal/auth/app/refresh_api_key"
	"github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/internal/auth/fixture"
	"github.com/YuukanOO/seelf/internal/auth/infra/crypto"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/assert"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/bus/spy"
)

func Test_RefreshApiKey(t *testing.T) {

	arrange := func(tb testing.TB, seed ...fixture.SeedBuilder) (
		bus.RequestHandler[string, refresh_api_key.Command],
		spy.Dispatcher,
	) {
		context := fixture.PrepareDatabase(tb, seed...)
		return refresh_api_key.Handler(context.UsersStore, context.UsersStore, crypto.NewKeyGenerator()), context.Dispatcher
	}

	t.Run("should fail if the user does not exists", func(t *testing.T) {
		handler, _ := arrange(t)

		_, err := handler(context.Background(), refresh_api_key.Command{})

		assert.ErrorIs(t, apperr.ErrNotFound, err)
	})

	t.Run("should refresh the user's API key if everything is good", func(t *testing.T) {
		existingUser := fixture.User()
		handler, dispatcher := arrange(t, fixture.WithUsers(&existingUser))

		key, err := handler(context.Background(), refresh_api_key.Command{
			ID: string(existingUser.ID())},
		)

		assert.Nil(t, err)
		assert.NotEqual(t, "", key)

		assert.HasLength(t, 1, dispatcher.Signals())
		keyChanged := assert.Is[domain.UserAPIKeyChanged](t, dispatcher.Signals()[0])

		assert.Equal(t, domain.UserAPIKeyChanged{
			ID:  existingUser.ID(),
			Key: domain.APIKey(key),
		}, keyChanged)
	})
}
