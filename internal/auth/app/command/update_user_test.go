package command_test

import (
	"context"
	"testing"

	"github.com/YuukanOO/seelf/internal/auth/app/command"
	"github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/internal/auth/infra"
	"github.com/YuukanOO/seelf/internal/auth/infra/memory"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

func Test_UpdateUser(t *testing.T) {
	hasher := infra.NewBCryptHasher()
	update := func(existingUsers ...domain.User) (func(context.Context, command.UpdateUserCommand) error, memory.UsersStore) {
		store := memory.NewUsersStore(existingUsers...)
		return command.UpdateUser(store, store, hasher), store
	}

	t.Run("should require valid inputs", func(t *testing.T) {
		uc, _ := update()
		err := uc(context.Background(), command.UpdateUserCommand{})

		testutil.ErrorIs(t, apperr.ErrNotFound, err)
	})

	t.Run("should succeed if values are the same", func(t *testing.T) {
		passwordHash, _ := hasher.Hash("apassword")
		user := domain.NewUser("john@doe.com", passwordHash, "anapikey")
		uc, store := update(user)

		err := uc(context.Background(), command.UpdateUserCommand{
			ID:       string(user.ID()),
			Email:    monad.Value("john@doe.com"),
			Password: monad.Value("apassword"),
		})

		testutil.IsNil(t, err)

		user, _ = store.GetByID(context.Background(), user.ID())
		testutil.HasNEvents(t, &user, 2) // 2 since bcrypt will produce different hashes
	})

	t.Run("should update user if everything is good", func(t *testing.T) {
		user := domain.NewUser("john@doe.com", "apassword", "anapikey")
		uc, store := update(user)

		err := uc(context.Background(), command.UpdateUserCommand{
			ID:       string(user.ID()),
			Email:    monad.Value("another@email.com"),
			Password: monad.Value("anotherpassword"),
		})

		testutil.IsNil(t, err)

		user, _ = store.GetByID(context.Background(), user.ID())
		testutil.HasNEvents(t, &user, 3)
	})
}
