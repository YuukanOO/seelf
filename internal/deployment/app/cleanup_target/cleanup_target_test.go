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
)

func Test_CleanupTarget(t *testing.T) {

	arrange := func(tb testing.TB, provider domain.Provider, seed ...fixture.SeedBuilder) (
		bus.RequestHandler[bus.UnitType, cleanup_target.Command],
		context.Context,
	) {
		context := fixture.PrepareDatabase(tb, seed...)
		return cleanup_target.Handler(context.TargetsStore, context.DeploymentsStore, provider), context.Context
	}

	t.Run("should silently fail if the target does not exist anymore", func(t *testing.T) {
		var provider dummyProvider
		handler, ctx := arrange(t, &provider)

		_, err := handler(ctx, cleanup_target.Command{})

		assert.Nil(t, err)
		assert.False(t, provider.called)
	})

	t.Run("should skip the cleanup if the target has never been configured correctly", func(t *testing.T) {
		var provider dummyProvider
		user := authfixture.User()
		target := fixture.Target(fixture.WithTargetCreatedBy(user.ID()))
		target.Configured(target.CurrentVersion(), nil, errors.New("configuration_failed"))
		handler, ctx := arrange(t, &provider,
			fixture.WithUsers(&user),
			fixture.WithTargets(&target),
		)

		_, err := handler(ctx, cleanup_target.Command{
			ID: string(target.ID()),
		})

		assert.Nil(t, err)
		assert.False(t, provider.called)
	})

	t.Run("should fail if a deployment is running on this target", func(t *testing.T) {
		var provider dummyProvider
		user := authfixture.User()
		target := fixture.Target(fixture.WithTargetCreatedBy(user.ID()))
		app := fixture.App(fixture.WithAppCreatedBy(user.ID()),
			fixture.WithEnvironmentConfig(
				domain.NewEnvironmentConfig(target.ID()),
				domain.NewEnvironmentConfig(target.ID()),
			))
		deployment := fixture.Deployment(fixture.FromApp(app),
			fixture.ForEnvironment(domain.Production),
			fixture.WithDeploymentRequestedBy(user.ID()))
		assert.Nil(t, deployment.HasStarted())
		handler, ctx := arrange(t, &provider,
			fixture.WithUsers(&user),
			fixture.WithTargets(&target),
			fixture.WithApps(&app),
			fixture.WithDeployments(&deployment),
		)

		_, err := handler(ctx, cleanup_target.Command{
			ID: string(target.ID()),
		})

		assert.ErrorIs(t, domain.ErrRunningOrPendingDeployments, err)
		assert.False(t, provider.called)
	})

	t.Run("should fail if being configured", func(t *testing.T) {
		var provider dummyProvider
		user := authfixture.User()
		target := fixture.Target(fixture.WithTargetCreatedBy(user.ID()))
		handler, ctx := arrange(t, &provider,
			fixture.WithUsers(&user),
			fixture.WithTargets(&target),
		)

		_, err := handler(ctx, cleanup_target.Command{
			ID: string(target.ID()),
		})

		assert.ErrorIs(t, domain.ErrTargetConfigurationInProgress, err)
		assert.False(t, provider.called)
	})

	t.Run("should fail if has been configured in the past but is now unreachable", func(t *testing.T) {
		var provider dummyProvider
		user := authfixture.User()
		target := fixture.Target(fixture.WithTargetCreatedBy(user.ID()))
		target.Configured(target.CurrentVersion(), nil, nil)
		assert.Nil(t, target.Reconfigure())
		target.Configured(target.CurrentVersion(), nil, errors.New("configuration_failed"))
		handler, ctx := arrange(t, &provider,
			fixture.WithUsers(&user),
			fixture.WithTargets(&target),
		)

		_, err := handler(ctx, cleanup_target.Command{
			ID: string(target.ID()),
		})

		assert.ErrorIs(t, domain.ErrTargetConfigurationFailed, err)
		assert.False(t, provider.called)
	})

	t.Run("should cleanup the target if it is correctly configured", func(t *testing.T) {
		var provider dummyProvider
		user := authfixture.User()
		target := fixture.Target(fixture.WithTargetCreatedBy(user.ID()))
		target.Configured(target.CurrentVersion(), nil, nil)
		handler, ctx := arrange(t, &provider,
			fixture.WithUsers(&user),
			fixture.WithTargets(&target),
		)

		_, err := handler(ctx, cleanup_target.Command{
			ID: string(target.ID()),
		})

		assert.Nil(t, err)
		assert.True(t, provider.called)
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
