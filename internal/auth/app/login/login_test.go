package login_test

import (
	"context"
	"testing"

	"github.com/YuukanOO/seelf/internal/auth/app/login"
	"github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/internal/auth/infra/crypto"
	"github.com/YuukanOO/seelf/internal/auth/infra/memory"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/must"
	"github.com/YuukanOO/seelf/pkg/testutil"
	"github.com/YuukanOO/seelf/pkg/validate"
)

func Test_Login(t *testing.T) {
	hasher := crypto.NewBCryptHasher()
	password := must.Panic(hasher.Hash("password")) // Sample password hash for the string "password" for tests
	existingUser := must.Panic(domain.NewUser(domain.NewEmailRequirement("existing@example.com", true), password, "apikey"))

	sut := func(existingUsers ...*domain.User) bus.RequestHandler[string, login.Command] {
		store := memory.NewUsersStore(existingUsers...)
		return login.Handler(store, hasher)
	}

	t.Run("should require valid inputs", func(t *testing.T) {
		uc := sut()
		_, err := uc(context.Background(), login.Command{})

		testutil.ErrorIs(t, validate.ErrValidationFailed, err)
	})

	t.Run("should complains if email does not exists", func(t *testing.T) {
		uc := sut()
		_, err := uc(context.Background(), login.Command{
			Email:    "notexisting@example.com",
			Password: "nobodycares",
		})

		validationErr, ok := apperr.As[validate.FieldErrors](err)
		testutil.IsTrue(t, ok)
		testutil.ErrorIs(t, domain.ErrInvalidEmailOrPassword, validationErr["email"])
		testutil.ErrorIs(t, domain.ErrInvalidEmailOrPassword, validationErr["password"])
	})

	t.Run("should complains if password does not match", func(t *testing.T) {
		uc := sut(&existingUser)
		_, err := uc(context.Background(), login.Command{
			Email:    "existing@example.com",
			Password: "nobodycares",
		})

		validationErr, ok := apperr.As[validate.FieldErrors](err)
		testutil.IsTrue(t, ok)
		testutil.ErrorIs(t, domain.ErrInvalidEmailOrPassword, validationErr["email"])
		testutil.ErrorIs(t, domain.ErrInvalidEmailOrPassword, validationErr["password"])
	})

	t.Run("should returns a valid user id if it succeeds", func(t *testing.T) {
		uc := sut(&existingUser)
		uid, err := uc(context.Background(), login.Command{
			Email:    "existing@example.com",
			Password: "password",
		})

		testutil.IsNil(t, err)
		testutil.Equals(t, string(existingUser.ID()), uid)
	})
}
