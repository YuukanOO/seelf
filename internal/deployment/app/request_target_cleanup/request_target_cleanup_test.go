package request_target_cleanup_test

import (
	"context"
	"testing"

	authfixture "github.com/YuukanOO/seelf/internal/auth/fixture"
	"github.com/YuukanOO/seelf/internal/deployment/app/request_target_cleanup"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/deployment/fixture"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/assert"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/bus/spy"
	shared "github.com/YuukanOO/seelf/pkg/domain"
)

func Test_RequestTargetCleanup(t *testing.T) {

	arrange := func(tb testing.TB, seed ...fixture.SeedBuilder) (
		bus.RequestHandler[bus.UnitType, request_target_cleanup.Command],
		context.Context,
		spy.Dispatcher,
	) {
		context := fixture.PrepareDatabase(tb, seed...)
		return request_target_cleanup.Handler(context.TargetsStore, context.TargetsStore, context.AppsStore), context.Context, context.Dispatcher
	}

	t.Run("should returns an error if the target does not exist", func(t *testing.T) {
		handler, ctx, _ := arrange(t)

		_, err := handler(ctx, request_target_cleanup.Command{
			ID: "some-id",
		})

		assert.ErrorIs(t, apperr.ErrNotFound, err)
	})

	t.Run("should returns an error if the target has still apps using it", func(t *testing.T) {
		user := authfixture.User()
		target := fixture.Target(fixture.WithTargetCreatedBy(user.ID()))
		target.Configured(target.CurrentVersion(), nil, nil)
		app := fixture.App(
			fixture.WithAppCreatedBy(user.ID()),
			fixture.WithEnvironmentConfig(
				domain.NewEnvironmentConfig(target.ID()),
				domain.NewEnvironmentConfig(target.ID()),
			),
		)
		handler, ctx, _ := arrange(t,
			fixture.WithUsers(&user),
			fixture.WithTargets(&target),
			fixture.WithApps(&app),
		)

		_, err := handler(ctx, request_target_cleanup.Command{
			ID: string(target.ID()),
		})

		assert.ErrorIs(t, domain.ErrTargetInUse, err)
	})

	t.Run("should returns an error if the target is configuring", func(t *testing.T) {
		user := authfixture.User()
		target := fixture.Target(fixture.WithTargetCreatedBy(user.ID()))
		handler, ctx, _ := arrange(t,
			fixture.WithUsers(&user),
			fixture.WithTargets(&target),
		)

		_, err := handler(ctx, request_target_cleanup.Command{
			ID: string(target.ID()),
		})

		assert.ErrorIs(t, domain.ErrTargetConfigurationInProgress, err)
	})

	t.Run("should correctly mark the target for cleanup", func(t *testing.T) {
		user := authfixture.User()
		target := fixture.Target(fixture.WithTargetCreatedBy(user.ID()))
		target.Configured(target.CurrentVersion(), nil, nil)
		handler, ctx, dispatcher := arrange(t,
			fixture.WithUsers(&user),
			fixture.WithTargets(&target),
		)

		_, err := handler(ctx, request_target_cleanup.Command{
			ID: string(target.ID()),
		})

		assert.Nil(t, err)
		assert.HasLength(t, 1, dispatcher.Signals())
		requested := assert.Is[domain.TargetCleanupRequested](t, dispatcher.Signals()[0])
		assert.Equal(t, domain.TargetCleanupRequested{
			ID:        target.ID(),
			Requested: shared.ActionFrom(user.ID(), assert.NotZero(t, requested.Requested.At())),
		}, requested)
	})
}
