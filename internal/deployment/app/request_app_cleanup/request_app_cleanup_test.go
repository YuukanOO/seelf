package request_app_cleanup_test

import (
	"context"
	"testing"

	authfixture "github.com/YuukanOO/seelf/internal/auth/fixture"
	"github.com/YuukanOO/seelf/internal/deployment/app/request_app_cleanup"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/deployment/fixture"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/assert"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/bus/spy"
	shared "github.com/YuukanOO/seelf/pkg/domain"
)

func Test_RequestAppCleanup(t *testing.T) {

	arrange := func(tb testing.TB, seed ...fixture.SeedBuilder) (
		bus.RequestHandler[bus.UnitType, request_app_cleanup.Command],
		context.Context,
		spy.Dispatcher,
	) {
		context := fixture.PrepareDatabase(tb, seed...)
		return request_app_cleanup.Handler(context.AppsStore, context.AppsStore), context.Context, context.Dispatcher
	}

	t.Run("should fail if the application does not exist", func(t *testing.T) {
		handler, ctx, _ := arrange(t)

		r, err := handler(ctx, request_app_cleanup.Command{
			ID: "some-id",
		})

		assert.ErrorIs(t, apperr.ErrNotFound, err)
		assert.Equal(t, bus.Unit, r)
	})

	t.Run("should mark an application has ready for deletion", func(t *testing.T) {
		user := authfixture.User()
		target := fixture.Target(fixture.WithTargetCreatedBy(user.ID()))
		app := fixture.App(
			fixture.WithAppCreatedBy(user.ID()),
			fixture.WithEnvironmentConfig(
				domain.NewEnvironmentConfig(target.ID()),
				domain.NewEnvironmentConfig(target.ID()),
			),
		)

		handler, ctx, dispatcher := arrange(t,
			fixture.WithUsers(&user),
			fixture.WithTargets(&target),
			fixture.WithApps(&app),
		)

		r, err := handler(ctx, request_app_cleanup.Command{
			ID: string(app.ID()),
		})

		assert.Nil(t, err)
		assert.Equal(t, bus.Unit, r)
		assert.HasLength(t, 1, dispatcher.Signals())
		requested := assert.Is[domain.AppCleanupRequested](t, dispatcher.Signals()[0])
		assert.DeepEqual(t, domain.AppCleanupRequested{
			ID:         app.ID(),
			Production: requested.Production,
			Staging:    requested.Staging,
			Requested:  shared.ActionFrom(user.ID(), assert.NotZero(t, requested.Requested.At())),
		}, requested)
	})
}
