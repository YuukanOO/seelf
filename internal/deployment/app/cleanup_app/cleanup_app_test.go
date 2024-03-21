package cleanup_app_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/YuukanOO/seelf/cmd/config"
	"github.com/YuukanOO/seelf/internal/deployment/app/cleanup_app"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/deployment/infra/artifact"
	"github.com/YuukanOO/seelf/internal/deployment/infra/memory"
	"github.com/YuukanOO/seelf/internal/deployment/infra/source/raw"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/log"
	"github.com/YuukanOO/seelf/pkg/must"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

type initialData struct {
	apps        []*domain.App
	deployments []*domain.Deployment
	targets     []*domain.Target
}

func Test_CleanupApp(t *testing.T) {
	ctx := context.Background()
	logger, _ := log.NewLogger()

	sut := func(initialData initialData) (bus.RequestHandler[bus.UnitType, cleanup_app.Command], *dummyProvider) {
		opts := config.Default(config.WithTestDefaults())
		appsStore := memory.NewAppsStore(initialData.apps...)
		targetsStore := memory.NewTargetsStore(initialData.targets...)
		deploymentsStore := memory.NewDeploymentsStore(initialData.deployments...)
		artifactManager := artifact.NewLocal(opts, logger)

		t.Cleanup(func() {
			os.RemoveAll(opts.DataDir())
		})
		provider := &dummyProvider{}
		return cleanup_app.Handler(deploymentsStore, appsStore, appsStore, artifactManager, provider, targetsStore), provider
	}

	t.Run("should fail silently if the application does not exist anymore", func(t *testing.T) {
		uc, provider := sut(initialData{})

		r, err := uc(ctx, cleanup_app.Command{
			ID: "some-id",
		})

		testutil.Equals(t, bus.Ignore(apperr.ErrNotFound), err)
		testutil.Equals(t, bus.Unit, r)
		testutil.IsFalse(t, provider.called)
	})

	t.Run("should fail if the application cleanup has not been requested", func(t *testing.T) {
		app := must.Panic(domain.NewApp("my-app", domain.NewEnvironmentConfig("1"), domain.NewEnvironmentConfig("1"), domain.AppNamingProductionAvailable|domain.AppNamingStagingAvailable, "uid"))
		uc, provider := sut(initialData{
			apps: []*domain.App{&app},
		})

		r, err := uc(ctx, cleanup_app.Command{
			ID: string(app.ID()),
		})

		testutil.ErrorIs(t, domain.ErrAppCleanupNeeded, err)
		testutil.Equals(t, bus.Unit, r)
		testutil.IsFalse(t, provider.called)
	})

	t.Run("should fail if they are still running or pending deployments", func(t *testing.T) {
		app := must.Panic(domain.NewApp("my-app", domain.NewEnvironmentConfig("1"), domain.NewEnvironmentConfig("1"), domain.AppNamingProductionAvailable|domain.AppNamingStagingAvailable, "uid"))
		depl := must.Panic(app.NewDeployment(1, raw.Data(""), domain.Production, "uid"))
		app.RequestCleanup("uid")

		uc, provider := sut(initialData{
			apps:        []*domain.App{&app},
			deployments: []*domain.Deployment{&depl},
		})

		r, err := uc(ctx, cleanup_app.Command{
			ID: string(app.ID()),
		})

		testutil.ErrorIs(t, domain.ErrAppHasRunningOrPendingDeployments, err)
		testutil.Equals(t, bus.Unit, r)
		testutil.IsFalse(t, provider.called)
	})

	t.Run("should fail if the target is configuring", func(t *testing.T) {
		target := must.Panic(domain.NewTarget("my-target", must.Panic(domain.UrlFrom("http://localhost")), true, nil, true, "uid"))

		app := must.Panic(domain.NewApp("my-app",
			domain.NewEnvironmentConfig(target.ID()),
			domain.NewEnvironmentConfig(target.ID()), domain.AppNamingProductionAvailable|domain.AppNamingStagingAvailable, "uid"))
		app.RequestCleanup("uid")

		uc, provider := sut(initialData{
			apps:    []*domain.App{&app},
			targets: []*domain.Target{&target},
		})

		_, err := uc(ctx, cleanup_app.Command{
			ID: string(app.ID()),
		})

		testutil.ErrorIs(t, domain.ErrTargetConfigurationInProgress, err)
		testutil.IsFalse(t, provider.called)
	})

	t.Run("should succeed if the target is being deleted", func(t *testing.T) {
		target := must.Panic(domain.NewTarget("my-target", must.Panic(domain.UrlFrom("http://localhost")), true, nil, true, "uid"))
		created := testutil.EventIs[domain.TargetCreated](t, &target, 0)
		target.Configured(created.State.Version(), nil)
		target.RequestDelete(0, "uid")

		app := must.Panic(domain.NewApp("my-app",
			domain.NewEnvironmentConfig(target.ID()),
			domain.NewEnvironmentConfig(target.ID()), domain.AppNamingProductionAvailable|domain.AppNamingStagingAvailable, "uid"))
		app.RequestCleanup("uid")

		uc, provider := sut(initialData{
			apps:    []*domain.App{&app},
			targets: []*domain.Target{&target},
		})

		_, err := uc(ctx, cleanup_app.Command{
			ID: string(app.ID()),
		})

		testutil.IsNil(t, err)
		testutil.EventIs[domain.AppDeleted](t, &app, 2)
		testutil.IsFalse(t, provider.called)
	})

	t.Run("should fail if the target configuration has failed but has been reachable is the past", func(t *testing.T) {
		target := must.Panic(domain.NewTarget("my-target", must.Panic(domain.UrlFrom("http://localhost")), true, nil, true, "uid"))
		created := testutil.EventIs[domain.TargetCreated](t, &target, 0)
		target.Configured(created.State.Version(), nil)
		target.Reconfigure()
		changed := testutil.EventIs[domain.TargetStateChanged](t, &target, 1)
		target.Configured(changed.State.Version(), errors.New("configuration-failed"))

		app := must.Panic(domain.NewApp("my-app",
			domain.NewEnvironmentConfig(target.ID()),
			domain.NewEnvironmentConfig(target.ID()), domain.AppNamingProductionAvailable|domain.AppNamingStagingAvailable, "uid"))
		app.RequestCleanup("uid")

		uc, provider := sut(initialData{
			apps:    []*domain.App{&app},
			targets: []*domain.Target{&target},
		})

		_, err := uc(ctx, cleanup_app.Command{
			ID: string(app.ID()),
		})

		testutil.ErrorIs(t, domain.ErrTargetConfigurationFailed, err)
		testutil.IsFalse(t, provider.called)
	})

	t.Run("should succeed if the target configuration has failed but has never been reachable", func(t *testing.T) {
		target := must.Panic(domain.NewTarget("my-target", must.Panic(domain.UrlFrom("http://localhost")), true, nil, true, "uid"))
		created := testutil.EventIs[domain.TargetCreated](t, &target, 0)
		target.Configured(created.State.Version(), errors.New("configuration-failed"))

		app := must.Panic(domain.NewApp("my-app",
			domain.NewEnvironmentConfig(target.ID()),
			domain.NewEnvironmentConfig(target.ID()), domain.AppNamingProductionAvailable|domain.AppNamingStagingAvailable, "uid"))
		app.RequestCleanup("uid")

		uc, provider := sut(initialData{
			apps:    []*domain.App{&app},
			targets: []*domain.Target{&target},
		})

		_, err := uc(ctx, cleanup_app.Command{
			ID: string(app.ID()),
		})

		testutil.IsNil(t, err)
		testutil.EventIs[domain.AppDeleted](t, &app, 2)
		testutil.IsFalse(t, provider.called)
	})

	t.Run("should succeed if the target does not exist anymore", func(t *testing.T) {
		app := must.Panic(domain.NewApp("my-app", domain.NewEnvironmentConfig("1"), domain.NewEnvironmentConfig("1"), domain.AppNamingProductionAvailable|domain.AppNamingStagingAvailable, "uid"))
		app.RequestCleanup("uid")

		uc, provider := sut(initialData{
			apps: []*domain.App{&app},
		})

		r, err := uc(ctx, cleanup_app.Command{
			ID: string(app.ID()),
		})

		testutil.IsNil(t, err)
		testutil.Equals(t, bus.Unit, r)
		testutil.HasNEvents(t, &app, 3)
		testutil.EventIs[domain.AppDeleted](t, &app, 2)
		testutil.IsFalse(t, provider.called)
	})

	t.Run("should succeed if the target exist and is correctly configured", func(t *testing.T) {
		target := must.Panic(domain.NewTarget("my-target", must.Panic(domain.UrlFrom("http://localhost")), true, nil, true, "uid"))
		created := testutil.EventIs[domain.TargetCreated](t, &target, 0)
		target.Configured(created.State.Version(), nil)

		app := must.Panic(domain.NewApp("my-app",
			domain.NewEnvironmentConfig(target.ID()),
			domain.NewEnvironmentConfig(target.ID()), domain.AppNamingProductionAvailable|domain.AppNamingStagingAvailable, "uid"))
		app.RequestCleanup("uid")

		uc, provider := sut(initialData{
			apps:    []*domain.App{&app},
			targets: []*domain.Target{&target},
		})

		r, err := uc(ctx, cleanup_app.Command{
			ID: string(app.ID()),
		})

		testutil.IsNil(t, err)
		testutil.Equals(t, bus.Unit, r)
		testutil.HasNEvents(t, &app, 3)
		testutil.EventIs[domain.AppDeleted](t, &app, 2)
		testutil.IsTrue(t, provider.called)
	})
}

type dummyProvider struct {
	domain.Provider
	called bool
}

func (d *dummyProvider) Cleanup(context.Context, domain.AppID, domain.Target, domain.Environment) error {
	d.called = true
	return nil
}
