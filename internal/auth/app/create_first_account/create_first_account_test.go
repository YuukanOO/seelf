package create_first_account_test

import (
	"context"
	"testing"

	"github.com/YuukanOO/seelf/internal/auth/app/create_first_account"
	"github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/internal/auth/fixture"
	"github.com/YuukanOO/seelf/internal/auth/infra/crypto"
	"github.com/YuukanOO/seelf/pkg/assert"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/bus/spy"
	"github.com/YuukanOO/seelf/pkg/validate"
)

func Test_CreateFirstAccount(t *testing.T) {

	arrange := func(tb testing.TB, seed ...fixture.SeedBuilder) (
		bus.RequestHandler[string, create_first_account.Command],
		spy.Dispatcher,
	) {
		context := fixture.PrepareDatabase(tb, seed...)
		return create_first_account.Handler(context.UsersStore, context.UsersStore, crypto.NewBCryptHasher(), crypto.NewKeyGenerator()), context.Dispatcher
	}

	t.Run("should returns the existing user id if a user already exists", func(t *testing.T) {
		existingUser := fixture.User()
		handler, dispatcher := arrange(t, fixture.WithUsers(&existingUser))

		uid, err := handler(context.Background(), create_first_account.Command{})

		assert.Nil(t, err)
		assert.Equal(t, string(existingUser.ID()), uid)
		assert.HasLength(t, 0, dispatcher.Signals())
	})

	t.Run("should require both email and password or fail with ErrAdminAccountRequired", func(t *testing.T) {
		handler, _ := arrange(t)
		uid, err := handler(context.Background(), create_first_account.Command{})

		assert.ErrorIs(t, create_first_account.ErrAdminAccountRequired, err)
		assert.Equal(t, "", uid)

		uid, err = handler(context.Background(), create_first_account.Command{Email: "admin@example.com"})
		assert.ErrorIs(t, create_first_account.ErrAdminAccountRequired, err)
		assert.Equal(t, "", uid)

		uid, err = handler(context.Background(), create_first_account.Command{Password: "admin"})
		assert.ErrorIs(t, create_first_account.ErrAdminAccountRequired, err)
		assert.Equal(t, "", uid)
	})

	t.Run("should require valid inputs", func(t *testing.T) {
		handler, _ := arrange(t)
		uid, err := handler(context.Background(), create_first_account.Command{
			Email:    "not_an_email",
			Password: "admin",
		})

		assert.Equal(t, "", uid)
		assert.ValidationError(t, validate.FieldErrors{
			"email": domain.ErrInvalidEmail,
		}, err)
	})

	t.Run("should creates the first user account if everything is good", func(t *testing.T) {
		handler, dispatcher := arrange(t)
		uid, err := handler(context.Background(), create_first_account.Command{
			Email:    "admin@example.com",
			Password: "admin",
		})

		assert.Nil(t, err)
		assert.NotEqual(t, "", uid)

		assert.HasLength(t, 1, dispatcher.Signals())
		registered := assert.Is[domain.UserRegistered](t, dispatcher.Signals()[0])

		assert.Equal(t, domain.UserRegistered{
			ID:           domain.UserID(uid),
			Email:        "admin@example.com",
			Password:     assert.NotZero(t, registered.Password),
			RegisteredAt: assert.NotZero(t, registered.RegisteredAt),
			Key:          assert.NotZero(t, registered.Key),
		}, registered)
	})
}
