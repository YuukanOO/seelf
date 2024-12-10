package cleanup_app_test

import (
	"context"
	"errors"
	"testing"
	"time"

	authfixture "github.com/YuukanOO/seelf/internal/auth/fixture"
	"github.com/YuukanOO/seelf/internal/deployment/app/cleanup_app"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/deployment/fixture"
	"github.com/YuukanOO/seelf/pkg/assert"
	"github.com/YuukanOO/seelf/pkg/bus"
)

func Test_CleanupApp(t *testing.T) {

	arrange := func(tb testing.TB, provider domain.Provider, seed ...fixture.SeedBuilder) (
		bus.RequestHandler[bus.AsyncResult, cleanup_app.Command],
		context.Context,
	) {
		context := fixture.PrepareDatabase(tb, seed...)
		return cleanup_app.Handler(context.TargetsStore, context.DeploymentsStore, context.AppsStore, context.AppsStore, provider, context.UnitOfWorkFactory), context.Context
	}

	t.Run("should fail silently if the target does not exist anymore", func(t *testing.T) {
		var provider mockProvider
		handler, ctx := arrange(t, &provider)

		r, err := handler(ctx, cleanup_app.Command{})

		assert.Nil(t, err)
		assert.Equal(t, bus.AsyncResultProcessed, r)
		assert.False(t, provider.called)
	})

	t.Run("should delay if at least one deployment is running", func(t *testing.T) {
		var provider mockProvider
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

		result, err := handler(ctx, cleanup_app.Command{
			TargetID:    string(target.ID()),
			AppID:       string(app.ID()),
			Environment: string(domain.Production),
			From:        deployment.Requested().At().Add(-1 * time.Hour),
			To:          deployment.Requested().At().Add(1 * time.Hour),
		})

		assert.Equal(t, bus.AsyncResultDelay, result)
		assert.Nil(t, err)
		assert.False(t, provider.called)
	})

	t.Run("should delay if the target is configuring and at least one successful deployment has been made", func(t *testing.T) {
		var provider mockProvider
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
		assert.Nil(t, deployment.HasEnded(domain.Services{}, nil))

		handler, ctx := arrange(t, &provider,
			fixture.WithUsers(&user),
			fixture.WithTargets(&target),
			fixture.WithApps(&app),
			fixture.WithDeployments(&deployment),
		)

		result, err := handler(ctx, cleanup_app.Command{
			TargetID:    string(target.ID()),
			AppID:       string(app.ID()),
			Environment: string(domain.Production),
			From:        deployment.Requested().At().Add(-1 * time.Hour),
			To:          deployment.Requested().At().Add(1 * time.Hour),
		})

		assert.Equal(t, bus.AsyncResultDelay, result)
		assert.Nil(t, err)
		assert.False(t, provider.called)
	})

	t.Run("should succeed if the target is being deleted", func(t *testing.T) {
		var provider mockProvider
		user := authfixture.User()
		target := fixture.Target(fixture.WithTargetCreatedBy(user.ID()))
		assert.Nil(t, target.Configured(target.CurrentVersion(), nil, nil))
		assert.Nil(t, target.RequestDelete(false, "uid"))

		handler, ctx := arrange(t, &provider,
			fixture.WithUsers(&user),
			fixture.WithTargets(&target),
		)

		_, err := handler(ctx, cleanup_app.Command{
			TargetID: string(target.ID()),
		})

		assert.Nil(t, err)
		assert.False(t, provider.called)
	})

	t.Run("should succeed if no successful deployments has been made", func(t *testing.T) {
		var provider mockProvider
		user := authfixture.User()
		target := fixture.Target(fixture.WithTargetCreatedBy(user.ID()))

		handler, ctx := arrange(t, &provider,
			fixture.WithUsers(&user),
			fixture.WithTargets(&target),
		)

		_, err := handler(ctx, cleanup_app.Command{
			TargetID: string(target.ID()),
		})

		assert.Nil(t, err)
		assert.False(t, provider.called)
	})

	t.Run("should fail if the target is not ready and a successful deployment has been made", func(t *testing.T) {
		var provider mockProvider
		user := authfixture.User()
		target := fixture.Target(fixture.WithTargetCreatedBy(user.ID()))
		assert.Nil(t, target.Configured(target.CurrentVersion(), nil, errors.New("configuration failed")))
		app := fixture.App(fixture.WithAppCreatedBy(user.ID()),
			fixture.WithEnvironmentConfig(
				domain.NewEnvironmentConfig(target.ID()),
				domain.NewEnvironmentConfig(target.ID()),
			))
		deployment := fixture.Deployment(fixture.FromApp(app),
			fixture.ForEnvironment(domain.Production),
			fixture.WithDeploymentRequestedBy(user.ID()))
		assert.Nil(t, deployment.HasStarted())
		assert.Nil(t, deployment.HasEnded(domain.Services{}, nil))

		handler, ctx := arrange(t, &provider,
			fixture.WithUsers(&user),
			fixture.WithTargets(&target),
			fixture.WithApps(&app),
			fixture.WithDeployments(&deployment),
		)

		_, err := handler(ctx, cleanup_app.Command{
			TargetID:    string(target.ID()),
			AppID:       string(app.ID()),
			Environment: string(domain.Production),
			From:        deployment.Requested().At().Add(-1 * time.Hour),
			To:          deployment.Requested().At().Add(1 * time.Hour),
		})

		assert.ErrorIs(t, domain.ErrTargetConfigurationFailed, err)
		assert.False(t, provider.called)
	})

	t.Run("should succeed if the target is ready and successful deployments have been made", func(t *testing.T) {
		var provider mockProvider
		user := authfixture.User()
		target := fixture.Target(fixture.WithTargetCreatedBy(user.ID()))
		assert.Nil(t, target.Configured(target.CurrentVersion(), nil, nil))
		app := fixture.App(fixture.WithAppCreatedBy(user.ID()),
			fixture.WithEnvironmentConfig(
				domain.NewEnvironmentConfig(target.ID()),
				domain.NewEnvironmentConfig(target.ID()),
			))
		deployment := fixture.Deployment(fixture.FromApp(app),
			fixture.ForEnvironment(domain.Production),
			fixture.WithDeploymentRequestedBy(user.ID()))
		assert.Nil(t, deployment.HasStarted())
		assert.Nil(t, deployment.HasEnded(domain.Services{}, nil))

		handler, ctx := arrange(t, &provider,
			fixture.WithUsers(&user),
			fixture.WithTargets(&target),
			fixture.WithApps(&app),
			fixture.WithDeployments(&deployment),
		)

		_, err := handler(ctx, cleanup_app.Command{
			TargetID:    string(target.ID()),
			AppID:       string(app.ID()),
			Environment: string(domain.Production),
			From:        deployment.Requested().At().Add(-1 * time.Hour),
			To:          deployment.Requested().At().Add(1 * time.Hour),
		})

		assert.Nil(t, err)
		assert.True(t, provider.called)
	})
}

type mockProvider struct {
	domain.Provider
	called bool
}

func (d *mockProvider) Cleanup(_ context.Context, _ domain.AppID, _ domain.Target, _ domain.Environment, s domain.CleanupStrategy) error {
	d.called = s != domain.CleanupStrategySkip
	return nil
}
