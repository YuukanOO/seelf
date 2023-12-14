package infra

import (
	"context"

	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/log"
	"github.com/YuukanOO/seelf/pkg/storage/sqlite"

	"github.com/YuukanOO/seelf/internal/auth/app/create_first_account"
	"github.com/YuukanOO/seelf/internal/auth/app/login"
	"github.com/YuukanOO/seelf/internal/auth/app/update_user"
	"github.com/YuukanOO/seelf/internal/auth/domain"
	authsqlite "github.com/YuukanOO/seelf/internal/auth/infra/sqlite"
)

type Options interface {
	DefaultEmail() string
	DefaultPassword() string
}

// Setup the auth module
func Setup(
	opts Options,
	logger log.Logger,
	db sqlite.Database,
	b bus.Bus,
) (domain.UsersReader, error) {
	usersStore := authsqlite.NewUsersStore(db)
	authQueryHandler := authsqlite.NewGateway(db)

	passwordHasher := NewBCryptHasher()
	keyGenerator := NewKeyGenerator()

	bus.Register(b, login.Handler(usersStore, passwordHasher))
	bus.Register(b, create_first_account.Handler(usersStore, usersStore, passwordHasher, keyGenerator))
	bus.Register(b, update_user.Handler(usersStore, usersStore, passwordHasher))
	bus.Register(b, authQueryHandler.GetProfile)

	if err := db.Migrate(authsqlite.Migrations); err != nil {
		return nil, err
	}

	// Create the first account if needed
	if _, err := bus.Send(b, context.Background(), create_first_account.Command{
		Email:    opts.DefaultEmail(),
		Password: opts.DefaultPassword(),
	}); err != nil {
		return nil, err
	}

	return usersStore, nil
}
