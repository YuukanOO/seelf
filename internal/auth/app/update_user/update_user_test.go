package update_user_test

import (
	"context"
	"testing"

	"github.com/YuukanOO/seelf/internal/auth/app/update_user"
	"github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/internal/auth/infra/crypto"
	"github.com/YuukanOO/seelf/internal/auth/infra/memory"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/must"
	"github.com/YuukanOO/seelf/pkg/testutil"
	"github.com/YuukanOO/seelf/pkg/validate"
)

func Test_UpdateUser(t *testing.T) {
	hasher := crypto.NewBCryptHasher()
	passwordHash := must.Panic(hasher.Hash("apassword"))

	sut := func(existingUsers ...*domain.User) bus.RequestHandler[string, update_user.Command] {
		store := memory.NewUsersStore(existingUsers...)
		return update_user.Handler(store, store, hasher)
	}

	t.Run("should require valid inputs", func(t *testing.T) {
		uc := sut()
		_, err := uc(context.Background(), update_user.Command{})

		testutil.ErrorIs(t, apperr.ErrNotFound, err)
	})

	t.Run("should fail if the email is taken by another user", func(t *testing.T) {
		john := must.Panic(domain.NewUser(domain.NewEmailRequirement("john@doe.com", true), passwordHash, "anapikey"))
		jane := must.Panic(domain.NewUser(domain.NewEmailRequirement("jane@doe.com", true), passwordHash, "anapikey"))

		uc := sut(&john, &jane)

		_, err := uc(context.Background(), update_user.Command{
			ID:    string(john.ID()),
			Email: monad.Value("jane@doe.com"),
		})

		validationErr, ok := apperr.As[validate.FieldErrors](err)
		testutil.IsTrue(t, ok)
		testutil.ErrorIs(t, domain.ErrEmailAlreadyTaken, validationErr["email"])
	})

	t.Run("should succeed if values are the same", func(t *testing.T) {
		john := must.Panic(domain.NewUser(domain.NewEmailRequirement("john@doe.com", true), passwordHash, "anapikey"))
		uc := sut(&john)

		id, err := uc(context.Background(), update_user.Command{
			ID:       string(john.ID()),
			Email:    monad.Value("john@doe.com"),
			Password: monad.Value("apassword"),
		})

		testutil.IsNil(t, err)
		testutil.Equals(t, string(john.ID()), id)
		testutil.HasNEvents(t, &john, 2) // 2 since bcrypt will produce different hashes
		testutil.EventIs[domain.UserPasswordChanged](t, &john, 1)
	})

	t.Run("should update user if everything is good", func(t *testing.T) {
		john := must.Panic(domain.NewUser(domain.NewEmailRequirement("john@doe.com", true), passwordHash, "anapikey"))
		uc := sut(&john)

		id, err := uc(context.Background(), update_user.Command{
			ID:       string(john.ID()),
			Email:    monad.Value("another@email.com"),
			Password: monad.Value("anotherpassword"),
		})

		testutil.IsNil(t, err)
		testutil.Equals(t, string(john.ID()), id)
		testutil.HasNEvents(t, &john, 3)
		evt := testutil.EventIs[domain.UserEmailChanged](t, &john, 1)
		testutil.Equals(t, "another@email.com", string(evt.Email))
		testutil.EventIs[domain.UserPasswordChanged](t, &john, 2)
	})
}
