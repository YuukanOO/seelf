package command_test

import (
	"context"
	"testing"

	"github.com/YuukanOO/seelf/internal/auth/app/command"
	"github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/internal/auth/infra"
	"github.com/YuukanOO/seelf/internal/auth/infra/memory"
	"github.com/YuukanOO/seelf/pkg/testutil"
	"github.com/YuukanOO/seelf/pkg/validation"
)

func Test_Register(t *testing.T) {
	register := func(existingUsers ...domain.User) func(context.Context, command.RegisterCommand) (string, error) {
		store := memory.NewUsersStore(existingUsers...)
		hasher := infra.NewBCryptHasher()
		keygen := infra.NewKeyGenerator()
		return command.Register(store, store, hasher, keygen)
	}

	t.Run("should require valid inputs", func(t *testing.T) {
		uc := register()
		_, err := uc(context.Background(), command.RegisterCommand{})

		testutil.ErrorIs(t, validation.ErrValidationFailed, err)
	})

	t.Run("should fail if a user with the same email already exists", func(t *testing.T) {
		uc := register(domain.NewUser("existing@example.com", "password", "apikey"))
		_, err := uc(context.Background(), command.RegisterCommand{
			Email:    "existing@example.com",
			Password: "nobodycares",
		})

		testutil.ErrorIs(t, domain.ErrEmailAlreadyTaken, err)
	})

	t.Run("should succeed if everything is good", func(t *testing.T) {
		uc := register()
		uid, err := uc(context.Background(), command.RegisterCommand{
			Email:    "anemail@example.com",
			Password: "nobodycares",
		})

		testutil.IsNil(t, err)
		testutil.NotEquals(t, "", uid)
	})
}
