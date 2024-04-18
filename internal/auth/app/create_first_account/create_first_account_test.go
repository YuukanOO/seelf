package create_first_account_test

import (
	"context"
	"testing"

	"github.com/YuukanOO/seelf/internal/auth/app/create_first_account"
	"github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/internal/auth/infra/crypto"
	"github.com/YuukanOO/seelf/internal/auth/infra/memory"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/must"
	"github.com/YuukanOO/seelf/pkg/testutil"
	"github.com/YuukanOO/seelf/pkg/validate"
)

func Test_CreateFirstAccount(t *testing.T) {
	ctx := context.Background()
	hasher := crypto.NewBCryptHasher()
	keygen := crypto.NewKeyGenerator()

	sut := func(existingUsers ...*domain.User) bus.RequestHandler[string, create_first_account.Command] {
		store := memory.NewUsersStore(existingUsers...)
		return create_first_account.Handler(store, store, hasher, keygen)
	}

	t.Run("should returns the existing user id if a user already exists", func(t *testing.T) {
		usr := must.Panic(domain.NewUser(domain.NewEmailRequirement("existing@example.com", true), "password", "apikey"))
		uc := sut(&usr)

		uid, err := uc(ctx, create_first_account.Command{})

		testutil.IsNil(t, err)
		testutil.Equals(t, string(usr.ID()), uid)
	})

	t.Run("should require both email and password or fail with ErrAdminAccountRequired", func(t *testing.T) {
		uc := sut()
		uid, err := uc(ctx, create_first_account.Command{})

		testutil.ErrorIs(t, create_first_account.ErrAdminAccountRequired, err)
		testutil.Equals(t, "", uid)

		uid, err = uc(ctx, create_first_account.Command{Email: "admin@example.com"})
		testutil.ErrorIs(t, create_first_account.ErrAdminAccountRequired, err)
		testutil.Equals(t, "", uid)

		uid, err = uc(ctx, create_first_account.Command{Password: "admin"})
		testutil.ErrorIs(t, create_first_account.ErrAdminAccountRequired, err)
		testutil.Equals(t, "", uid)

	})

	t.Run("should require valid inputs", func(t *testing.T) {
		uc := sut()
		uid, err := uc(ctx, create_first_account.Command{
			Email:    "notanemail",
			Password: "admin",
		})

		testutil.ErrorIs(t, validate.ErrValidationFailed, err)
		testutil.Equals(t, "", uid)

	})

	t.Run("should creates the first user account if everything is good", func(t *testing.T) {
		uc := sut()
		uid, err := uc(ctx, create_first_account.Command{
			Email:    "admin@example.com",
			Password: "admin",
		})

		testutil.IsNil(t, err)
		testutil.NotEquals(t, "", uid)
	})
}
