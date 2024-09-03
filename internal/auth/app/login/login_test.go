package login_test

import (
	"context"
	"testing"

	"github.com/YuukanOO/seelf/internal/auth/app/login"
	"github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/internal/auth/fixture"
	"github.com/YuukanOO/seelf/internal/auth/infra/crypto"
	"github.com/YuukanOO/seelf/pkg/assert"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/bus/spy"
	"github.com/YuukanOO/seelf/pkg/validate"
	"github.com/YuukanOO/seelf/pkg/validate/strings"
)

func Test_Login(t *testing.T) {
	hasher := crypto.NewBCryptHasher()

	arrange := func(tb testing.TB, seed ...fixture.SeedBuilder) (
		bus.RequestHandler[string, login.Command],
		spy.Dispatcher,
	) {
		context := fixture.PrepareDatabase(tb, seed...)
		return login.Handler(context.UsersStore, hasher), context.Dispatcher
	}

	t.Run("should require valid inputs", func(t *testing.T) {
		handler, _ := arrange(t)
		_, err := handler(context.Background(), login.Command{})

		assert.ValidationError(t, validate.FieldErrors{
			"email":    domain.ErrInvalidEmail,
			"password": strings.ErrRequired,
		}, err)
	})

	t.Run("should complains if email does not exists", func(t *testing.T) {
		handler, _ := arrange(t)
		_, err := handler(context.Background(), login.Command{
			Email:    "notexisting@example.com",
			Password: "no_body_cares",
		})

		assert.ValidationError(t, validate.FieldErrors{
			"email":    domain.ErrInvalidEmailOrPassword,
			"password": domain.ErrInvalidEmailOrPassword,
		}, err)
	})

	t.Run("should complains if password does not match", func(t *testing.T) {
		existingUser := fixture.User(
			fixture.WithEmail("existing@example.com"),
			fixture.WithPassword("raw_password_hash", hasher),
		)
		handler, _ := arrange(t, fixture.WithUsers(&existingUser))

		_, err := handler(context.Background(), login.Command{
			Email:    "existing@example.com",
			Password: "no_body_cares",
		})

		assert.ValidationError(t, validate.FieldErrors{
			"email":    domain.ErrInvalidEmailOrPassword,
			"password": domain.ErrInvalidEmailOrPassword,
		}, err)
	})

	t.Run("should returns a valid user id if it succeeds", func(t *testing.T) {
		existingUser := fixture.User(
			fixture.WithEmail("existing@example.com"),
			fixture.WithPassword("password", hasher),
		)
		handler, dispatcher := arrange(t, fixture.WithUsers(&existingUser))

		uid, err := handler(context.Background(), login.Command{
			Email:    "existing@example.com",
			Password: "password",
		})

		assert.Nil(t, err)
		assert.Equal(t, string(existingUser.ID()), uid)
		assert.HasLength(t, 0, dispatcher.Signals())
	})
}
