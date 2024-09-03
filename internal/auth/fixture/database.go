//go:build !release

package fixture

import (
	"context"
	"os"
	"testing"

	"github.com/YuukanOO/seelf/cmd/config"
	"github.com/YuukanOO/seelf/internal/auth/domain"
	auth "github.com/YuukanOO/seelf/internal/auth/infra/sqlite"
	"github.com/YuukanOO/seelf/pkg/bus/spy"
	"github.com/YuukanOO/seelf/pkg/log"
	"github.com/YuukanOO/seelf/pkg/must"
	"github.com/YuukanOO/seelf/pkg/ostools"
	"github.com/YuukanOO/seelf/pkg/storage/sqlite"
)

type (
	seed struct {
		users []*domain.User
	}

	Context struct {
		Context    context.Context // If users has been seeded, will be authenticated as the first one
		Dispatcher spy.Dispatcher
		UsersStore auth.UsersStore
	}

	SeedBuilder func(*seed)
)

func PrepareDatabase(t testing.TB, options ...SeedBuilder) *Context {
	cfg := config.Default(config.WithTestDefaults())

	if err := ostools.MkdirAll(cfg.DataDir()); err != nil {
		t.Fatal(err)
	}

	result := Context{
		Context:    context.Background(),
		Dispatcher: spy.NewDispatcher(),
	}

	db, err := sqlite.Open(cfg.ConnectionString(), must.Panic(log.NewLogger()), result.Dispatcher)

	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		db.Close()
		os.RemoveAll(cfg.DataDir())
	})

	if err = db.Migrate(auth.Migrations); err != nil {
		t.Fatal(err)
	}

	result.UsersStore = auth.NewUsersStore(db)

	// Seed the database
	var s seed

	for _, o := range options {
		o(&s)
	}

	if err := result.UsersStore.Write(result.Context, s.users...); err != nil {
		t.Fatal(err)
	}

	if len(s.users) > 0 {
		result.Context = domain.WithUserID(result.Context, s.users[0].ID()) // The first created user will be used as the authenticated one
	}

	// Reset the dispatcher after seeding
	result.Dispatcher.Reset()

	return &result
}

func WithUsers(users ...*domain.User) SeedBuilder {
	return func(s *seed) {
		s.users = users
	}
}
