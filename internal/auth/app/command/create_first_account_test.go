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

func Test_CreateFirstAccount(t *testing.T) {
	ctx := context.Background()
	hasher := infra.NewBCryptHasher()
	keygen := infra.NewKeyGenerator()

	createFirstAccount := func(existingUsers ...*domain.User) (func(context.Context, command.CreateFirstAccountCommand) error, memory.UsersStore) {
		store := memory.NewUsersStore(existingUsers...)
		return command.CreateFirstAccount(store, store, hasher, keygen), store
	}

	t.Run("should do nothing if a user already exists", func(t *testing.T) {
		usr := domain.NewUser("existing@example.com", "password", "apikey")
		uc, store := createFirstAccount(&usr)

		err := uc(ctx, command.CreateFirstAccountCommand{})

		testutil.IsNil(t, err)

		count, err := store.GetUsersCount(ctx)
		testutil.IsNil(t, err)
		testutil.Equals(t, 1, count)
	})

	t.Run("should require both email and password or fail with ErrAdminAccountRequired", func(t *testing.T) {
		uc, _ := createFirstAccount()
		err := uc(ctx, command.CreateFirstAccountCommand{})

		testutil.ErrorIs(t, domain.ErrAdminAccountRequired, err)

		err = uc(ctx, command.CreateFirstAccountCommand{Email: "admin@example.com"})
		testutil.ErrorIs(t, domain.ErrAdminAccountRequired, err)

		err = uc(ctx, command.CreateFirstAccountCommand{Password: "admin"})
		testutil.ErrorIs(t, domain.ErrAdminAccountRequired, err)
	})

	t.Run("should require valid inputs", func(t *testing.T) {
		uc, _ := createFirstAccount()
		err := uc(ctx, command.CreateFirstAccountCommand{
			Email:    "notanemail",
			Password: "admin",
		})

		testutil.ErrorIs(t, validation.ErrValidationFailed, err)
	})

	t.Run("should creates the first user account if everything is good", func(t *testing.T) {
		uc, store := createFirstAccount()
		err := uc(ctx, command.CreateFirstAccountCommand{
			Email:    "admin@example.com",
			Password: "admin",
		})

		testutil.IsNil(t, err)
		count, _ := store.GetUsersCount(ctx)
		testutil.Equals(t, 1, count)
	})
}
