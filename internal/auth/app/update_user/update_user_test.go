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
	"github.com/YuukanOO/seelf/pkg/testutil"
)

func Test_UpdateUser(t *testing.T) {
	hasher := crypto.NewBCryptHasher()

	sut := func(existingUsers ...*domain.User) bus.RequestHandler[string, update_user.Command] {
		store := memory.NewUsersStore(existingUsers...)
		return update_user.Handler(store, store, hasher)
	}

	t.Run("should require valid inputs", func(t *testing.T) {
		uc := sut()
		_, err := uc(context.Background(), update_user.Command{})

		testutil.ErrorIs(t, apperr.ErrNotFound, err)
	})

	t.Run("should succeed if values are the same", func(t *testing.T) {
		passwordHash, _ := hasher.Hash("apassword")
		user := domain.NewUser("john@doe.com", passwordHash, "anapikey")
		uc := sut(&user)

		id, err := uc(context.Background(), update_user.Command{
			ID:       string(user.ID()),
			Email:    monad.Value("john@doe.com"),
			Password: monad.Value("apassword"),
		})

		testutil.IsNil(t, err)
		testutil.Equals(t, string(user.ID()), id)
		testutil.HasNEvents(t, &user, 2) // 2 since bcrypt will produce different hashes
		testutil.EventIs[domain.UserPasswordChanged](t, &user, 1)
	})

	t.Run("should update user if everything is good", func(t *testing.T) {
		user := domain.NewUser("john@doe.com", "apassword", "anapikey")
		uc := sut(&user)

		id, err := uc(context.Background(), update_user.Command{
			ID:       string(user.ID()),
			Email:    monad.Value("another@email.com"),
			Password: monad.Value("anotherpassword"),
		})

		testutil.IsNil(t, err)
		testutil.Equals(t, string(user.ID()), id)
		testutil.HasNEvents(t, &user, 3)
		evt := testutil.EventIs[domain.UserEmailChanged](t, &user, 1)
		testutil.Equals(t, "another@email.com", string(evt.Email))
		testutil.EventIs[domain.UserPasswordChanged](t, &user, 2)
	})
}
