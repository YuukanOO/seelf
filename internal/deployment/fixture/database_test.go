package fixture_test

import (
	"testing"

	auth "github.com/YuukanOO/seelf/internal/auth/domain"
	authfixture "github.com/YuukanOO/seelf/internal/auth/fixture"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/deployment/fixture"
	"github.com/YuukanOO/seelf/pkg/assert"
)

func Test_Database(t *testing.T) {
	t.Run("should be able to prepare a database without seeding it", func(t *testing.T) {
		ctx := fixture.PrepareDatabase(t)

		assert.NotNil(t, ctx)
		assert.NotNil(t, ctx.Config)
		assert.NotNil(t, ctx.AppsStore)
		assert.NotNil(t, ctx.TargetsStore)
		assert.NotNil(t, ctx.AppsStore)
		assert.NotNil(t, ctx.DeploymentsStore)
		assert.NotNil(t, ctx.RegistriesStore)
		assert.NotNil(t, ctx.Dispatcher)
		assert.HasLength(t, 0, ctx.Dispatcher.Signals())
		assert.HasLength(t, 0, ctx.Dispatcher.Requests())
	})

	t.Run("should seed correctly and attach the first user id to the created context", func(t *testing.T) {
		user := authfixture.User()
		target := fixture.Target(fixture.WithTargetCreatedBy(user.ID()))
		config := domain.NewEnvironmentConfig(target.ID())
		app := fixture.App(
			fixture.WithEnvironmentConfig(config, config),
			fixture.WithAppCreatedBy(user.ID()),
		)
		registry := fixture.Registry(fixture.WithRegistryCreatedBy(user.ID()))
		deployment := fixture.Deployment(
			fixture.FromApp(app),
			fixture.WithDeploymentRequestedBy(user.ID()),
		)

		ctx := fixture.PrepareDatabase(t,
			fixture.WithUsers(&user),
			fixture.WithTargets(&target),
			fixture.WithApps(&app),
			fixture.WithRegistries(&registry),
			fixture.WithDeployments(&deployment),
		)

		assert.Equal(t, user.ID(), auth.CurrentUser(ctx.Context).Get(""))
	})
}
