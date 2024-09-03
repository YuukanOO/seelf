//go:build !release

package fixture

import (
	"context"
	"os"
	"testing"

	"github.com/YuukanOO/seelf/cmd/config"
	auth "github.com/YuukanOO/seelf/internal/auth/domain"
	authsqlite "github.com/YuukanOO/seelf/internal/auth/infra/sqlite"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	deployment "github.com/YuukanOO/seelf/internal/deployment/infra/sqlite"
	"github.com/YuukanOO/seelf/pkg/bus/spy"
	scheduler "github.com/YuukanOO/seelf/pkg/bus/sqlite"
	"github.com/YuukanOO/seelf/pkg/log"
	"github.com/YuukanOO/seelf/pkg/must"
	"github.com/YuukanOO/seelf/pkg/ostools"
	"github.com/YuukanOO/seelf/pkg/storage/sqlite"
)

type (
	seed struct {
		users       []*auth.User
		targets     []*domain.Target
		apps        []*domain.App
		deployments []*domain.Deployment
		registries  []*domain.Registry
	}

	Context struct {
		Config           config.Configuration
		Context          context.Context // If users has been seeded, will be authenticated as the first one
		Dispatcher       spy.Dispatcher
		TargetsStore     deployment.TargetsStore
		AppsStore        deployment.AppsStore
		DeploymentsStore deployment.DeploymentsStore
		RegistriesStore  deployment.RegistriesStore
	}

	SeedBuilder func(*seed)
)

func PrepareDatabase(t testing.TB, options ...SeedBuilder) *Context {
	result := Context{
		Config:     config.Default(config.WithTestDefaults()),
		Context:    context.Background(),
		Dispatcher: spy.NewDispatcher(),
	}

	if err := ostools.MkdirAll(result.Config.DataDir()); err != nil {
		t.Fatal(err)
	}

	db, err := sqlite.Open(result.Config.ConnectionString(), must.Panic(log.NewLogger()), result.Dispatcher)

	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		db.Close()
		os.RemoveAll(result.Config.DataDir())
	})

	// FIXME: scheduler migrations are needed because some migrations may queue a job by inserting inside
	// the scheduled_jobs table. That's a mistake from my side and I should fix it later.
	if err = db.Migrate(scheduler.Migrations, authsqlite.Migrations, deployment.Migrations); err != nil {
		t.Fatal(err)
	}

	result.AppsStore = deployment.NewAppsStore(db)
	result.TargetsStore = deployment.NewTargetsStore(db)
	result.DeploymentsStore = deployment.NewDeploymentsStore(db)
	result.RegistriesStore = deployment.NewRegistriesStore(db)

	// Seed the database
	var s seed

	for _, o := range options {
		o(&s)
	}

	if len(s.users) > 0 {
		if err := authsqlite.NewUsersStore(db).Write(result.Context, s.users...); err != nil {
			t.Fatal(err)
		}
		result.Context = auth.WithUserID(result.Context, s.users[0].ID()) // The first created user will be used as the authenticated one
	}

	if err := result.RegistriesStore.Write(result.Context, s.registries...); err != nil {
		t.Fatal(err)
	}

	if err := result.TargetsStore.Write(result.Context, s.targets...); err != nil {
		t.Fatal(err)
	}

	if err := result.AppsStore.Write(result.Context, s.apps...); err != nil {
		t.Fatal(err)
	}

	if err := result.DeploymentsStore.Write(result.Context, s.deployments...); err != nil {
		t.Fatal(err)
	}

	// Reset the dispatcher after seeding
	result.Dispatcher.Reset()

	return &result
}

func WithUsers(users ...*auth.User) SeedBuilder {
	return func(s *seed) {
		s.users = users
	}
}

func WithTargets(targets ...*domain.Target) SeedBuilder {
	return func(s *seed) {
		s.targets = targets
	}
}

func WithApps(apps ...*domain.App) SeedBuilder {
	return func(s *seed) {
		s.apps = apps
	}
}

func WithDeployments(deployments ...*domain.Deployment) SeedBuilder {
	return func(s *seed) {
		s.deployments = deployments
	}
}

func WithRegistries(registries ...*domain.Registry) SeedBuilder {
	return func(s *seed) {
		s.registries = registries
	}
}
