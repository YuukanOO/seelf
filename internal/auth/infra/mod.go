package infra

import (
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/log"
	"github.com/YuukanOO/seelf/pkg/storage/sqlite"

	"github.com/YuukanOO/seelf/internal/auth/app/create_first_account"
	"github.com/YuukanOO/seelf/internal/auth/app/login"
	"github.com/YuukanOO/seelf/internal/auth/app/refresh_api_key"
	"github.com/YuukanOO/seelf/internal/auth/app/update_user"
	"github.com/YuukanOO/seelf/internal/auth/infra/crypto"
	authsqlite "github.com/YuukanOO/seelf/internal/auth/infra/sqlite"
)

// Setup the auth module
func Setup(
	logger log.Logger,
	db *sqlite.Database,
	b bus.Bus,
) error {
	usersStore := authsqlite.NewUsersStore(db)
	gateway := authsqlite.NewGateway(db)

	passwordHasher := crypto.NewBCryptHasher()
	keyGenerator := crypto.NewKeyGenerator()

	bus.Register(b, login.Handler(usersStore, passwordHasher))
	bus.Register(b, create_first_account.Handler(usersStore, usersStore, passwordHasher, keyGenerator))
	bus.Register(b, update_user.Handler(usersStore, usersStore, passwordHasher))
	bus.Register(b, refresh_api_key.Handler(usersStore, usersStore, keyGenerator))
	bus.Register(b, gateway.GetIDFromAPIKey)
	bus.Register(b, gateway.GetProfile)

	return db.Migrate(authsqlite.Migrations)
}
