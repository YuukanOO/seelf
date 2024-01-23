package cleanup_app_test

import (
	"context"
	"testing"
	"time"

	"github.com/YuukanOO/seelf/internal/deployment/app/cleanup_app"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/deployment/infra/memory"
	"github.com/YuukanOO/seelf/internal/deployment/infra/source/raw"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/must"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

type initialData struct {
	deployments []*domain.Deployment
	targets     []*domain.Target
}

func Test_CleanupApp(t *testing.T) {
	ctx := context.Background()

	sut := func(data initialData) (bus.RequestHandler[bus.UnitType, cleanup_app.Command], *dummyProvider) {
		targetsStore := memory.NewTargetsStore(data.targets...)
		deploymentsStore := memory.NewDeploymentsStore(data.deployments...)
		provider := &dummyProvider{}
		return cleanup_app.Handler(targetsStore, deploymentsStore, provider), provider
	}

	t.Run("should fail silently if the target does not exist anymore", func(t *testing.T) {
		uc, provider := sut(initialData{})

		r, err := uc(ctx, cleanup_app.Command{})

		testutil.IsNil(t, err)
		testutil.Equals(t, bus.Unit, r)
		testutil.IsFalse(t, provider.called)
	})

	t.Run("should fail if the target is configuring", func(t *testing.T) {
		target := must.Panic(domain.NewTarget("my-target",
			domain.NewTargetUrlRequirement(must.Panic(domain.UrlFrom("http://localhost")), true),
			domain.NewProviderConfigRequirement(nil, true), "uid"))
		app := must.Panic(domain.NewApp("my-app",
			domain.NewEnvironmentConfigRequirement(domain.NewEnvironmentConfig(target.ID()), true, true),
			domain.NewEnvironmentConfigRequirement(domain.NewEnvironmentConfig(target.ID()), true, true), "uid"))
		deployment := must.Panic(app.NewDeployment(1, raw.Data(""), domain.Production, "uid"))
		deployment.HasStarted()
		deployment.HasEnded(domain.Services{}, nil)

		uc, provider := sut(initialData{
			targets:     []*domain.Target{&target},
			deployments: []*domain.Deployment{&deployment},
		})

		_, err := uc(ctx, cleanup_app.Command{
			TargetID:    string(target.ID()),
			AppID:       string(app.ID()),
			Environment: string(domain.Production),
			From:        deployment.Requested().At().Add(-1 * time.Hour),
			To:          deployment.Requested().At().Add(1 * time.Hour),
		})

		testutil.ErrorIs(t, domain.ErrTargetConfigurationInProgress, err)
		testutil.IsFalse(t, provider.called)
	})

	t.Run("should succeed if the target is being deleted", func(t *testing.T) {
		target := must.Panic(domain.NewTarget("my-target",
			domain.NewTargetUrlRequirement(must.Panic(domain.UrlFrom("http://localhost")), true),
			domain.NewProviderConfigRequirement(nil, true), "uid"))
		target.Configured(target.CurrentVersion(), nil)
		target.RequestCleanup(false, "uid")

		uc, provider := sut(initialData{
			targets: []*domain.Target{&target},
		})

		_, err := uc(ctx, cleanup_app.Command{
			TargetID: string(target.ID()),
		})

		testutil.IsNil(t, err)
		testutil.IsFalse(t, provider.called)
	})

	t.Run("should succeed if no successful deployments has been made", func(t *testing.T) {
		target := must.Panic(domain.NewTarget("my-target",
			domain.NewTargetUrlRequirement(must.Panic(domain.UrlFrom("http://localhost")), true),
			domain.NewProviderConfigRequirement(nil, true), "uid"))
		target.Configured(target.CurrentVersion(), nil)

		uc, provider := sut(initialData{
			targets: []*domain.Target{&target},
		})

		_, err := uc(ctx, cleanup_app.Command{
			TargetID: string(target.ID()),
		})

		testutil.IsNil(t, err)
		testutil.IsFalse(t, provider.called)
	})

	t.Run("should succeed if the target is ready and successful deployments have been made", func(t *testing.T) {
		target := must.Panic(domain.NewTarget("my-target",
			domain.NewTargetUrlRequirement(must.Panic(domain.UrlFrom("http://localhost")), true),
			domain.NewProviderConfigRequirement(nil, true), "uid"))
		target.Configured(target.CurrentVersion(), nil)
		app := must.Panic(domain.NewApp("my-app",
			domain.NewEnvironmentConfigRequirement(domain.NewEnvironmentConfig(target.ID()), true, true),
			domain.NewEnvironmentConfigRequirement(domain.NewEnvironmentConfig(target.ID()), true, true), "uid"))
		deployment := must.Panic(app.NewDeployment(1, raw.Data(""), domain.Production, "uid"))
		deployment.HasStarted()
		deployment.HasEnded(domain.Services{}, nil)

		uc, provider := sut(initialData{
			targets:     []*domain.Target{&target},
			deployments: []*domain.Deployment{&deployment},
		})

		_, err := uc(ctx, cleanup_app.Command{
			TargetID:    string(target.ID()),
			AppID:       string(app.ID()),
			Environment: string(domain.Production),
			From:        deployment.Requested().At().Add(-1 * time.Hour),
			To:          deployment.Requested().At().Add(1 * time.Hour),
		})

		testutil.IsNil(t, err)
		testutil.IsTrue(t, provider.called)
	})
}

type dummyProvider struct {
	domain.Provider
	called bool
}

func (d *dummyProvider) Cleanup(_ context.Context, _ domain.AppID, _ domain.Target, _ domain.Environment, s domain.CleanupStrategy) error {
	d.called = s != domain.CleanupStrategySkip
	return nil
}
