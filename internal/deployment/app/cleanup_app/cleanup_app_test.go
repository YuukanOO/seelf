package cleanup_app_test

import (
	"context"
	"os"
	"testing"

	"github.com/YuukanOO/seelf/cmd/config"
	"github.com/YuukanOO/seelf/internal/deployment/app/cleanup_app"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/deployment/infra/artifact"
	"github.com/YuukanOO/seelf/internal/deployment/infra/memory"
	"github.com/YuukanOO/seelf/internal/deployment/infra/source/raw"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/log"
	"github.com/YuukanOO/seelf/pkg/must"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

type initialData struct {
	apps        []*domain.App
	deployments []*domain.Deployment
}

func Test_CleanupApp(t *testing.T) {
	ctx := context.Background()
	logger, _ := log.NewLogger()

	sut := func(initialData initialData) bus.RequestHandler[bus.UnitType, cleanup_app.Command] {
		opts := config.Default(config.WithTestDefaults())
		appsStore := memory.NewAppsStore(initialData.apps...)
		deploymentsStore := memory.NewDeploymentsStore(initialData.deployments...)
		artifactManager := artifact.NewLocal(opts, logger)

		t.Cleanup(func() {
			os.RemoveAll(opts.DataDir())
		})

		return cleanup_app.Handler(deploymentsStore, appsStore, appsStore, artifactManager, &dummyProvider{})
	}

	t.Run("should returns no error if the application does not exist", func(t *testing.T) {
		uc := sut(initialData{})

		r, err := uc(ctx, cleanup_app.Command{
			ID: "some-id",
		})

		testutil.IsNil(t, err)
		testutil.Equals(t, bus.Unit, r)
	})

	t.Run("should fail if the application cleanup as not been requested", func(t *testing.T) {
		app := must.Panic(domain.NewApp("my-app", domain.NewEnvironmentConfig("1"), domain.NewEnvironmentConfig("1"), domain.AppNamingProductionAvailable|domain.AppNamingStagingAvailable, "uid"))
		uc := sut(initialData{
			apps: []*domain.App{&app},
		})

		r, err := uc(ctx, cleanup_app.Command{
			ID: string(app.ID()),
		})

		testutil.ErrorIs(t, domain.ErrAppCleanupNeeded, err)
		testutil.Equals(t, bus.Unit, r)
	})

	t.Run("should fail if there are still pending or running deployments", func(t *testing.T) {
		app := must.Panic(domain.NewApp("my-app", domain.NewEnvironmentConfig("1"), domain.NewEnvironmentConfig("1"), domain.AppNamingProductionAvailable|domain.AppNamingStagingAvailable, "uid"))
		depl := must.Panic(app.NewDeployment(1, raw.Data(""), domain.Production, "uid"))
		app.RequestCleanup("uid")

		uc := sut(initialData{
			apps:        []*domain.App{&app},
			deployments: []*domain.Deployment{&depl},
		})

		r, err := uc(ctx, cleanup_app.Command{
			ID: string(app.ID()),
		})

		testutil.ErrorIs(t, domain.ErrAppHasRunningOrPendingDeployments, err)
		testutil.Equals(t, bus.Unit, r)
	})

	t.Run("should succeed if everything is good", func(t *testing.T) {
		app := must.Panic(domain.NewApp("my-app", domain.NewEnvironmentConfig("1"), domain.NewEnvironmentConfig("1"), domain.AppNamingProductionAvailable|domain.AppNamingStagingAvailable, "uid"))
		app.RequestCleanup("uid")

		uc := sut(initialData{
			apps: []*domain.App{&app},
		})

		r, err := uc(ctx, cleanup_app.Command{
			ID: string(app.ID()),
		})

		testutil.IsNil(t, err)
		testutil.Equals(t, bus.Unit, r)
		testutil.HasNEvents(t, &app, 3)
		evt := testutil.EventIs[domain.AppDeleted](t, &app, 2)
		testutil.Equals(t, app.ID(), evt.ID)
	})
}

type dummyProvider struct {
	domain.Provider
}

func (d *dummyProvider) Cleanup(ctx context.Context, app domain.App) error {
	return nil
}
