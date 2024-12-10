package cleanup_target_test

import (
	"context"
	"errors"
	"testing"

	authfixture "github.com/YuukanOO/seelf/internal/auth/fixture"
	"github.com/YuukanOO/seelf/internal/deployment/app/cleanup_target"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/deployment/fixture"
	"github.com/YuukanOO/seelf/pkg/assert"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/bus/spy"
)

func Test_CleanupTarget(t *testing.T) {

	arrange := func(tb testing.TB, provider domain.Provider, seed ...fixture.SeedBuilder) (
		bus.RequestHandler[bus.AsyncResult, cleanup_target.Command],
		context.Context,
		spy.Dispatcher,
	) {
		context := fixture.PrepareDatabase(tb, seed...)
		return cleanup_target.Handler(context.TargetsStore, context.TargetsStore, context.DeploymentsStore, provider, context.UnitOfWorkFactory), context.Context, context.Dispatcher
	}

	t.Run("should silently fail if the target does not exist anymore", func(t *testing.T) {
		var provider dummyProvider
		handler, ctx, _ := arrange(t, &provider)

		r, err := handler(ctx, cleanup_target.Command{})

		assert.Nil(t, err)
		assert.Equal(t, bus.AsyncResultProcessed, r)
		assert.False(t, provider.called)
	})

	t.Run("should fail if the target has not been requested for cleanup", func(t *testing.T) {
		var provider dummyProvider
		user := authfixture.User()
		target := fixture.Target(fixture.WithTargetCreatedBy(user.ID()))
		handler, ctx, _ := arrange(t, &provider,
			fixture.WithUsers(&user),
			fixture.WithTargets(&target),
		)

		r, err := handler(ctx, cleanup_target.Command{
			ID: string(target.ID()),
		})

		assert.ErrorIs(t, domain.ErrTargetCleanupNeeded, err)
		assert.Equal(t, bus.AsyncResultProcessed, r)
		assert.False(t, provider.called)
	})

	t.Run("should skip the cleanup if the target has never been configured correctly", func(t *testing.T) {
		var provider dummyProvider
		user := authfixture.User()
		target := fixture.Target(fixture.WithTargetCreatedBy(user.ID()))
		assert.Nil(t, target.Configured(target.CurrentVersion(), nil, errors.New("configuration_failed")))
		assert.Nil(t, target.RequestDelete(false, user.ID()))
		handler, ctx, _ := arrange(t, &provider,
			fixture.WithUsers(&user),
			fixture.WithTargets(&target),
		)

		r, err := handler(ctx, cleanup_target.Command{
			ID: string(target.ID()),
		})

		assert.Nil(t, err)
		assert.Equal(t, bus.AsyncResultProcessed, r)
		assert.False(t, provider.called)
	})

	t.Run("should be delayed if a deployment is running on this target", func(t *testing.T) {
		var provider dummyProvider
		user := authfixture.User()
		target := fixture.Target(fixture.WithTargetCreatedBy(user.ID()))
		assert.Nil(t, target.Configured(target.CurrentVersion(), nil, nil))
		assert.Nil(t, target.RequestDelete(false, user.ID()))
		app := fixture.App(fixture.WithAppCreatedBy(user.ID()),
			fixture.WithEnvironmentConfig(
				domain.NewEnvironmentConfig(target.ID()),
				domain.NewEnvironmentConfig(target.ID()),
			))
		deployment := fixture.Deployment(fixture.FromApp(app),
			fixture.ForEnvironment(domain.Production),
			fixture.WithDeploymentRequestedBy(user.ID()))
		assert.Nil(t, deployment.HasStarted())
		handler, ctx, _ := arrange(t, &provider,
			fixture.WithUsers(&user),
			fixture.WithTargets(&target),
			fixture.WithApps(&app),
			fixture.WithDeployments(&deployment),
		)

		r, err := handler(ctx, cleanup_target.Command{
			ID: string(target.ID()),
		})

		assert.Nil(t, err)
		assert.Equal(t, bus.AsyncResultDelay, r)
		assert.False(t, provider.called)
	})

	t.Run("should cleanup the target if it is correctly configured", func(t *testing.T) {
		var provider dummyProvider
		user := authfixture.User()
		target := fixture.Target(fixture.WithTargetCreatedBy(user.ID()))
		assert.Nil(t, target.Configured(target.CurrentVersion(), nil, nil))
		assert.Nil(t, target.RequestDelete(false, user.ID()))
		handler, ctx, dispatcher := arrange(t, &provider,
			fixture.WithUsers(&user),
			fixture.WithTargets(&target),
		)

		_, err := handler(ctx, cleanup_target.Command{
			ID: string(target.ID()),
		})

		assert.Nil(t, err)
		assert.True(t, provider.called)
		assert.HasLength(t, 1, dispatcher.Signals())
		deleted := assert.Is[domain.TargetDeleted](t, dispatcher.Signals()[0])
		assert.Equal(t, domain.TargetDeleted{
			ID: target.ID(),
		}, deleted)
	})
}

type dummyProvider struct {
	domain.Provider
	called bool
}

func (d *dummyProvider) CleanupTarget(_ context.Context, _ domain.Target, s domain.CleanupStrategy) error {
	d.called = s != domain.CleanupStrategySkip
	return nil
}
