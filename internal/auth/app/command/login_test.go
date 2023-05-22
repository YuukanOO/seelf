package command_test

import (
	"context"
	"testing"

	"github.com/YuukanOO/seelf/internal/auth/app/command"
	"github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/internal/auth/infra"
	"github.com/YuukanOO/seelf/internal/auth/infra/memory"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/testutil"
	"github.com/YuukanOO/seelf/pkg/validation"
)

func Test_Login(t *testing.T) {
	hasher := infra.NewBCryptHasher()
	password, _ := hasher.Hash("password") // Sample password hash for the string "password" for tests
	login := func(existingUsers ...domain.User) func(context.Context, command.LoginCommand) (string, error) {
		store := memory.NewUsersStore(existingUsers...)
		return command.Login(store, hasher)
	}

	t.Run("should require valid inputs", func(t *testing.T) {
		uc := login()
		_, err := uc(context.Background(), command.LoginCommand{})

		testutil.ErrorIs(t, validation.ErrValidationFailed, err)
	})

	t.Run("should complains if email does not exists", func(t *testing.T) {
		uc := login()
		_, err := uc(context.Background(), command.LoginCommand{
			Email:    "notexisting@example.com",
			Password: "nobodycares",
		})

		validationErr, ok := apperr.As[validation.Error](err)
		testutil.IsTrue(t, ok)
		testutil.ErrorIs(t, domain.ErrInvalidEmailOrPassword, validationErr.Fields["email"])
		testutil.ErrorIs(t, domain.ErrInvalidEmailOrPassword, validationErr.Fields["password"])
	})

	t.Run("should complains if password does not match", func(t *testing.T) {
		uc := login(domain.NewUser("existing@example.com", password, "apikey"))
		_, err := uc(context.Background(), command.LoginCommand{
			Email:    "existing@example.com",
			Password: "nobodycares",
		})

		validationErr, ok := apperr.As[validation.Error](err)
		testutil.IsTrue(t, ok)
		testutil.ErrorIs(t, domain.ErrInvalidEmailOrPassword, validationErr.Fields["email"])
		testutil.ErrorIs(t, domain.ErrInvalidEmailOrPassword, validationErr.Fields["password"])
	})

	t.Run("should returns a valid user id if it succeeds", func(t *testing.T) {
		existingUser := domain.NewUser("existing@example.com", password, "apikey")
		uc := login(existingUser)
		uid, err := uc(context.Background(), command.LoginCommand{
			Email:    "existing@example.com",
			Password: "password",
		})

		testutil.IsNil(t, err)
		testutil.Equals(t, string(existingUser.ID()), uid)
	})
}
