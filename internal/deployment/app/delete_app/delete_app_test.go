package delete_app_test

import (
	"context"
	"testing"

	authfixture "github.com/YuukanOO/seelf/internal/auth/fixture"
	"github.com/YuukanOO/seelf/internal/deployment/app/delete_app"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/deployment/fixture"
	"github.com/YuukanOO/seelf/internal/deployment/infra/artifact"
	"github.com/YuukanOO/seelf/pkg/assert"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/bus/spy"
	"github.com/YuukanOO/seelf/pkg/log"
)

func Test_DeleteApp(t *testing.T) {

	arrange := func(tb testing.TB, seed ...fixture.SeedBuilder) (
		bus.RequestHandler[bus.UnitType, delete_app.Command],
		spy.Dispatcher,
	) {
		context := fixture.PrepareDatabase(tb, seed...)
		logger, _ := log.NewLogger()
		artifactManager := artifact.NewLocal(context.Config, logger)
		return delete_app.Handler(context.AppsStore, context.AppsStore, artifactManager), context.Dispatcher
	}

	t.Run("should fail silently if the application does not exist anymore", func(t *testing.T) {
		handler, dispatcher := arrange(t)

		r, err := handler(context.Background(), delete_app.Command{
			ID: "some-id",
		})

		assert.Nil(t, err)
		assert.Equal(t, bus.Unit, r)
		assert.HasLength(t, 0, dispatcher.Signals())
	})

	t.Run("should fail if the application cleanup has not been requested first", func(t *testing.T) {
		user := authfixture.User()
		target := fixture.Target(fixture.WithTargetCreatedBy(user.ID()))
		app := fixture.App(
			fixture.WithAppCreatedBy(user.ID()),
			fixture.WithEnvironmentConfig(
				domain.NewEnvironmentConfig(target.ID()),
				domain.NewEnvironmentConfig(target.ID()),
			),
		)
		handler, _ := arrange(t,
			fixture.WithUsers(&user),
			fixture.WithTargets(&target),
			fixture.WithApps(&app),
		)

		r, err := handler(context.Background(), delete_app.Command{
			ID: string(app.ID()),
		})

		assert.ErrorIs(t, domain.ErrAppCleanupNeeded, err)
		assert.Equal(t, bus.Unit, r)
	})

	t.Run("should succeed if everything is good", func(t *testing.T) {
		user := authfixture.User()
		target := fixture.Target(fixture.WithTargetCreatedBy(user.ID()))
		app := fixture.App(
			fixture.WithAppCreatedBy(user.ID()),
			fixture.WithEnvironmentConfig(
				domain.NewEnvironmentConfig(target.ID()),
				domain.NewEnvironmentConfig(target.ID()),
			),
		)
		app.RequestCleanup(user.ID())
		handler, dispatcher := arrange(t,
			fixture.WithUsers(&user),
			fixture.WithTargets(&target),
			fixture.WithApps(&app),
		)

		r, err := handler(context.Background(), delete_app.Command{
			ID: string(app.ID()),
		})

		assert.Nil(t, err)
		assert.Equal(t, bus.Unit, r)
		assert.HasLength(t, 1, dispatcher.Signals())

		deleted := assert.Is[domain.AppDeleted](t, dispatcher.Signals()[0])
		assert.Equal(t, domain.AppDeleted{
			ID: app.ID(),
		}, deleted)
	})
}
