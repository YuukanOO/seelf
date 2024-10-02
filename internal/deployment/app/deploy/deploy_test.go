package deploy_test

import (
	"context"
	"errors"
	"testing"

	authfixture "github.com/YuukanOO/seelf/internal/auth/fixture"
	"github.com/YuukanOO/seelf/internal/deployment/app/deploy"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/deployment/fixture"
	"github.com/YuukanOO/seelf/internal/deployment/infra/artifact"
	"github.com/YuukanOO/seelf/internal/deployment/infra/source/raw"
	"github.com/YuukanOO/seelf/pkg/assert"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/bus/spy"
	"github.com/YuukanOO/seelf/pkg/log"
)

func Test_Deploy(t *testing.T) {

	arrange := func(
		tb testing.TB,
		source domain.Source,
		provider domain.Provider,
		seed ...fixture.SeedBuilder,
	) (
		bus.RequestHandler[bus.AsyncResult, deploy.Command],
		context.Context,
		spy.Dispatcher,
	) {
		context := fixture.PrepareDatabase(tb, seed...)
		logger, _ := log.NewLogger()
		artifactManager := artifact.NewLocal(context.Config, logger)
		return deploy.Handler(context.DeploymentsStore, context.DeploymentsStore, artifactManager, source, provider, context.TargetsStore, context.RegistriesStore), context.Context, context.Dispatcher
	}

	t.Run("should fail silently if the deployment does not exists", func(t *testing.T) {
		handler, ctx, _ := arrange(t, source(nil), provider(nil))

		r, err := handler(ctx, deploy.Command{})

		assert.Nil(t, err)
		assert.Equal(t, bus.AsyncResultProcessed, r)
	})

	t.Run("should delay the deployment if the target is configuring", func(t *testing.T) {
		user := authfixture.User()
		target := fixture.Target(fixture.WithTargetCreatedBy(user.ID()))
		app := fixture.App(
			fixture.WithAppCreatedBy(user.ID()),
			fixture.WithEnvironmentConfig(
				domain.NewEnvironmentConfig(target.ID()),
				domain.NewEnvironmentConfig(target.ID()),
			),
		)
		deployment := fixture.Deployment(
			fixture.WithDeploymentRequestedBy(user.ID()),
			fixture.FromApp(app),
		)
		handler, ctx, dispatcher := arrange(t, source(nil), provider(nil),
			fixture.WithUsers(&user),
			fixture.WithTargets(&target),
			fixture.WithApps(&app),
			fixture.WithDeployments(&deployment),
		)

		r, err := handler(ctx, deploy.Command{
			AppID:            string(deployment.ID().AppID()),
			DeploymentNumber: int(deployment.ID().DeploymentNumber()),
		})

		assert.Nil(t, err)
		assert.Equal(t, bus.AsyncResultDelay, r)
		assert.HasLength(t, 0, dispatcher.Signals())
	})

	t.Run("should mark the deployment has failed if source does not succeed", func(t *testing.T) {
		user := authfixture.User()
		target := fixture.Target(fixture.WithTargetCreatedBy(user.ID()))
		assert.Nil(t, target.Configured(target.CurrentVersion(), nil, nil))
		app := fixture.App(
			fixture.WithAppCreatedBy(user.ID()),
			fixture.WithEnvironmentConfig(
				domain.NewEnvironmentConfig(target.ID()),
				domain.NewEnvironmentConfig(target.ID()),
			),
		)
		deployment := fixture.Deployment(
			fixture.WithDeploymentRequestedBy(user.ID()),
			fixture.FromApp(app),
		)
		sourceErr := errors.New("source_failed")
		handler, ctx, dispatcher := arrange(t, source(sourceErr), provider(nil),
			fixture.WithUsers(&user),
			fixture.WithTargets(&target),
			fixture.WithApps(&app),
			fixture.WithDeployments(&deployment),
		)

		r, err := handler(ctx, deploy.Command{
			AppID:            string(deployment.ID().AppID()),
			DeploymentNumber: int(deployment.ID().DeploymentNumber()),
		})

		assert.Nil(t, err)
		assert.Equal(t, bus.AsyncResultProcessed, r)

		changed := assert.Is[domain.DeploymentStateChanged](t, dispatcher.Signals()[0])
		assert.Equal(t, domain.DeploymentStatusRunning, changed.State.Status())

		changed = assert.Is[domain.DeploymentStateChanged](t, dispatcher.Signals()[1])
		assert.Equal(t, domain.DeploymentStatusFailed, changed.State.Status())
		assert.Equal(t, sourceErr.Error(), changed.State.ErrCode().MustGet())
	})

	t.Run("should mark the deployment has failed if the target is not correctly configured", func(t *testing.T) {
		user := authfixture.User()
		target := fixture.Target(fixture.WithTargetCreatedBy(user.ID()))
		assert.Nil(t, target.Configured(target.CurrentVersion(), nil, errors.New("target_failed")))
		app := fixture.App(
			fixture.WithAppCreatedBy(user.ID()),
			fixture.WithEnvironmentConfig(
				domain.NewEnvironmentConfig(target.ID()),
				domain.NewEnvironmentConfig(target.ID()),
			),
		)
		deployment := fixture.Deployment(
			fixture.WithDeploymentRequestedBy(user.ID()),
			fixture.FromApp(app),
		)
		handler, ctx, dispatcher := arrange(t, source(nil), provider(nil),
			fixture.WithUsers(&user),
			fixture.WithTargets(&target),
			fixture.WithApps(&app),
			fixture.WithDeployments(&deployment),
		)

		r, err := handler(ctx, deploy.Command{
			AppID:            string(deployment.ID().AppID()),
			DeploymentNumber: int(deployment.ID().DeploymentNumber()),
		})

		assert.Nil(t, err)
		assert.Equal(t, bus.AsyncResultProcessed, r)

		changed := assert.Is[domain.DeploymentStateChanged](t, dispatcher.Signals()[0])
		assert.Equal(t, domain.DeploymentStatusRunning, changed.State.Status())

		changed = assert.Is[domain.DeploymentStateChanged](t, dispatcher.Signals()[1])
		assert.Equal(t, domain.DeploymentStatusFailed, changed.State.Status())
		assert.Equal(t, domain.ErrTargetConfigurationFailed.Error(), changed.State.ErrCode().MustGet())
	})

	t.Run("should mark the deployment has failed if provider does not run the deployment successfully", func(t *testing.T) {
		user := authfixture.User()
		target := fixture.Target(fixture.WithTargetCreatedBy(user.ID()))
		assert.Nil(t, target.Configured(target.CurrentVersion(), nil, nil))
		app := fixture.App(
			fixture.WithAppCreatedBy(user.ID()),
			fixture.WithEnvironmentConfig(
				domain.NewEnvironmentConfig(target.ID()),
				domain.NewEnvironmentConfig(target.ID()),
			),
		)
		deployment := fixture.Deployment(
			fixture.WithDeploymentRequestedBy(user.ID()),
			fixture.FromApp(app),
		)
		providerErr := errors.New("provider_failed")
		handler, ctx, dispatcher := arrange(t, source(nil), provider(providerErr),
			fixture.WithUsers(&user),
			fixture.WithTargets(&target),
			fixture.WithApps(&app),
			fixture.WithDeployments(&deployment),
		)

		r, err := handler(ctx, deploy.Command{
			AppID:            string(deployment.ID().AppID()),
			DeploymentNumber: int(deployment.ID().DeploymentNumber()),
		})

		assert.Nil(t, err)
		assert.Equal(t, bus.AsyncResultProcessed, r)

		changed := assert.Is[domain.DeploymentStateChanged](t, dispatcher.Signals()[0])
		assert.Equal(t, domain.DeploymentStatusRunning, changed.State.Status())

		changed = assert.Is[domain.DeploymentStateChanged](t, dispatcher.Signals()[1])
		assert.Equal(t, domain.DeploymentStatusFailed, changed.State.Status())
		assert.Equal(t, providerErr.Error(), changed.State.ErrCode().MustGet())
	})

	t.Run("should mark the deployment has succeeded if all is good", func(t *testing.T) {
		user := authfixture.User()
		target := fixture.Target(fixture.WithTargetCreatedBy(user.ID()))
		assert.Nil(t, target.Configured(target.CurrentVersion(), nil, nil))
		app := fixture.App(
			fixture.WithAppCreatedBy(user.ID()),
			fixture.WithEnvironmentConfig(
				domain.NewEnvironmentConfig(target.ID()),
				domain.NewEnvironmentConfig(target.ID()),
			),
		)
		deployment := fixture.Deployment(
			fixture.WithDeploymentRequestedBy(user.ID()),
			fixture.FromApp(app),
		)
		handler, ctx, dispatcher := arrange(t, source(nil), provider(nil),
			fixture.WithUsers(&user),
			fixture.WithTargets(&target),
			fixture.WithApps(&app),
			fixture.WithDeployments(&deployment),
		)

		r, err := handler(ctx, deploy.Command{
			AppID:            string(deployment.ID().AppID()),
			DeploymentNumber: int(deployment.ID().DeploymentNumber()),
		})

		assert.Nil(t, err)
		assert.Equal(t, bus.AsyncResultProcessed, r)

		changed := assert.Is[domain.DeploymentStateChanged](t, dispatcher.Signals()[0])
		assert.Equal(t, domain.DeploymentStatusRunning, changed.State.Status())

		changed = assert.Is[domain.DeploymentStateChanged](t, dispatcher.Signals()[1])
		assert.Equal(t, domain.DeploymentStatusSucceeded, changed.State.Status())
	})
}

type dummySource struct {
	err error
}

func source(failedWithErr error) domain.Source {
	return &dummySource{failedWithErr}
}

func (*dummySource) Prepare(context.Context, domain.App, any) (domain.SourceData, error) {
	return raw.Data(""), nil
}

func (t *dummySource) Fetch(context.Context, domain.DeploymentContext, domain.Deployment) error {
	return t.err
}

type dummyProvider struct {
	domain.Provider
	err error
}

func provider(failedWithErr error) domain.Provider {
	return &dummyProvider{
		err: failedWithErr,
	}
}

func (b *dummyProvider) Prepare(context.Context, any, ...domain.ProviderConfig) (domain.ProviderConfig, error) {
	return nil, nil
}

func (b *dummyProvider) Deploy(context.Context, domain.DeploymentContext, domain.Deployment, domain.Target, []domain.Registry) (domain.Services, error) {
	return domain.Services{}, b.err
}
