package update_user_test

import (
	"context"
	"testing"

	"github.com/YuukanOO/seelf/internal/auth/app/update_user"
	"github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/internal/auth/fixture"
	"github.com/YuukanOO/seelf/internal/auth/infra/crypto"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/assert"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/bus/spy"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/validate"
)

func Test_UpdateUser(t *testing.T) {

	arrange := func(tb testing.TB, seed ...fixture.SeedBuilder) (
		bus.RequestHandler[string, update_user.Command],
		spy.Dispatcher,
	) {
		context := fixture.PrepareDatabase(tb, seed...)
		return update_user.Handler(context.UsersStore, context.UsersStore, crypto.NewBCryptHasher()), context.Dispatcher
	}

	t.Run("should require an existing user", func(t *testing.T) {
		handler, _ := arrange(t)
		_, err := handler(context.Background(), update_user.Command{})

		assert.ErrorIs(t, apperr.ErrNotFound, err)
	})

	t.Run("should require valid inputs", func(t *testing.T) {
		handler, _ := arrange(t)

		_, err := handler(context.Background(), update_user.Command{
			Email: monad.Value("notanemail"),
		})

		assert.ValidationError(t, validate.FieldErrors{
			"email": domain.ErrInvalidEmail,
		}, err)
	})

	t.Run("should fail if the email is taken by another user", func(t *testing.T) {
		john := fixture.User(fixture.WithEmail("john@doe.com"))
		jane := fixture.User(fixture.WithEmail("jane@doe.com"))

		handler, _ := arrange(t, fixture.WithUsers(&john, &jane))

		_, err := handler(context.Background(), update_user.Command{
			ID:    string(john.ID()),
			Email: monad.Value("jane@doe.com"),
		})

		assert.ValidationError(t, validate.FieldErrors{
			"email": domain.ErrEmailAlreadyTaken,
		}, err)
	})

	t.Run("should succeed if values are the same", func(t *testing.T) {
		existingUser := fixture.User(fixture.WithEmail("john@doe.com"))
		handler, dispatcher := arrange(t, fixture.WithUsers(&existingUser))

		id, err := handler(context.Background(), update_user.Command{
			ID:       string(existingUser.ID()),
			Email:    monad.Value("john@doe.com"),
			Password: monad.Value("apassword"),
		})

		assert.Nil(t, err)
		assert.Equal(t, string(existingUser.ID()), id)

		assert.HasLength(t, 1, dispatcher.Signals())
		changed := assert.Is[domain.UserPasswordChanged](t, dispatcher.Signals()[0])

		assert.Equal(t, domain.UserPasswordChanged{
			ID:       existingUser.ID(),
			Password: changed.Password,
		}, changed)
	})

	t.Run("should update user if everything is good", func(t *testing.T) {
		existingUser := fixture.User()
		handler, dispatcher := arrange(t, fixture.WithUsers(&existingUser))

		id, err := handler(context.Background(), update_user.Command{
			ID:       string(existingUser.ID()),
			Email:    monad.Value("another@email.com"),
			Password: monad.Value("anotherpassword"),
		})

		assert.Nil(t, err)
		assert.Equal(t, string(existingUser.ID()), id)

		assert.HasLength(t, 2, dispatcher.Signals())

		passwordChanged := assert.Is[domain.UserPasswordChanged](t, dispatcher.Signals()[1])

		assert.Equal(t, domain.UserPasswordChanged{
			ID:       existingUser.ID(),
			Password: passwordChanged.Password,
		}, passwordChanged)

		emailChanged := assert.Is[domain.UserEmailChanged](t, dispatcher.Signals()[0])

		assert.Equal(t, domain.UserEmailChanged{
			ID:    existingUser.ID(),
			Email: "another@email.com",
		}, emailChanged)
	})
}
