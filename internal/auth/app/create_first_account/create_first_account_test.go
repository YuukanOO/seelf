package create_first_account_test

import (
	"context"
	"testing"

	"github.com/YuukanOO/seelf/internal/auth/app/create_first_account"
	"github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/internal/auth/infra/crypto"
	"github.com/YuukanOO/seelf/internal/auth/infra/memory"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/testutil"
	"github.com/YuukanOO/seelf/pkg/validation"
)

func Test_CreateFirstAccount(t *testing.T) {
	ctx := context.Background()
	hasher := crypto.NewBCryptHasher()
	keygen := crypto.NewKeyGenerator()

	sut := func(existingUsers ...*domain.User) (bus.RequestHandler[string, create_first_account.Command], memory.UsersStore) {
		store := memory.NewUsersStore(existingUsers...)
		return create_first_account.Handler(store, store, hasher, keygen), store
	}

	t.Run("should do nothing if a user already exists", func(t *testing.T) {
		usr := domain.NewUser("existing@example.com", "password", "apikey")
		uc, store := sut(&usr)

		uid, err := uc(ctx, create_first_account.Command{})

		testutil.IsNil(t, err)
		testutil.Equals(t, "", uid)

		count, err := store.GetUsersCount(ctx)
		testutil.IsNil(t, err)
		testutil.Equals(t, 1, count)
	})

	t.Run("should require both email and password or fail with ErrAdminAccountRequired", func(t *testing.T) {
		uc, _ := sut()
		uid, err := uc(ctx, create_first_account.Command{})

		testutil.ErrorIs(t, domain.ErrAdminAccountRequired, err)
		testutil.Equals(t, "", uid)

		uid, err = uc(ctx, create_first_account.Command{Email: "admin@example.com"})
		testutil.ErrorIs(t, domain.ErrAdminAccountRequired, err)
		testutil.Equals(t, "", uid)

		uid, err = uc(ctx, create_first_account.Command{Password: "admin"})
		testutil.ErrorIs(t, domain.ErrAdminAccountRequired, err)
		testutil.Equals(t, "", uid)

	})

	t.Run("should require valid inputs", func(t *testing.T) {
		uc, _ := sut()
		uid, err := uc(ctx, create_first_account.Command{
			Email:    "notanemail",
			Password: "admin",
		})

		testutil.ErrorIs(t, validation.ErrValidationFailed, err)
		testutil.Equals(t, "", uid)

	})

	t.Run("should creates the first user account if everything is good", func(t *testing.T) {
		uc, store := sut()
		uid, err := uc(ctx, create_first_account.Command{
			Email:    "admin@example.com",
			Password: "admin",
		})

		testutil.IsNil(t, err)
		testutil.NotEquals(t, "", uid)
		count, _ := store.GetUsersCount(ctx)
		testutil.Equals(t, 1, count)
	})
}
