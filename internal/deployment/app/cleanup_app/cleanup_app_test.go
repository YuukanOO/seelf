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
	"github.com/YuukanOO/seelf/pkg/testutil"
)

type initialData struct {
	existingApps        []*domain.App
	existingDeployments []*domain.Deployment
}

func Test_CleanupApp(t *testing.T) {
	ctx := context.Background()
	logger := log.NewLogger(false)

	sut := func(initialData initialData) bus.RequestHandler[bus.UnitType, cleanup_app.Command] {
		opts := config.Default(config.WithTestDefaults())
		appsStore := memory.NewAppsStore(initialData.existingApps...)
		deploymentsStore := memory.NewDeploymentsStore(initialData.existingDeployments...)
		artifactManager := artifact.NewLocal(opts, logger)

		t.Cleanup(func() {
			os.RemoveAll(opts.DataDir())
		})

		return cleanup_app.Handler(deploymentsStore, appsStore, appsStore, artifactManager, &dummyBackend{})
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
		a := domain.NewApp("my-app", "uid")
		uc := sut(initialData{
			existingApps: []*domain.App{&a},
		})

		r, err := uc(ctx, cleanup_app.Command{
			ID: string(a.ID()),
		})

		testutil.ErrorIs(t, domain.ErrAppCleanupNeeded, err)
		testutil.Equals(t, bus.Unit, r)
	})

	t.Run("should fail if there are still pending or running deployments", func(t *testing.T) {
		a := domain.NewApp("my-app", "uid")
		depl, _ := a.NewDeployment(1, raw.Data(""), domain.Production, "uid")
		a.RequestCleanup("uid")

		uc := sut(initialData{
			existingApps:        []*domain.App{&a},
			existingDeployments: []*domain.Deployment{&depl},
		})

		r, err := uc(ctx, cleanup_app.Command{
			ID: string(a.ID()),
		})

		testutil.ErrorIs(t, domain.ErrAppHasRunningOrPendingDeployments, err)
		testutil.Equals(t, bus.Unit, r)
	})

	t.Run("should succeed if everything is good", func(t *testing.T) {
		a := domain.NewApp("my-app", "uid")
		a.RequestCleanup("uid")

		uc := sut(initialData{
			existingApps: []*domain.App{&a},
		})

		r, err := uc(ctx, cleanup_app.Command{
			ID: string(a.ID()),
		})

		testutil.IsNil(t, err)
		testutil.Equals(t, bus.Unit, r)
		testutil.EventIs[domain.AppDeleted](t, &a, 2)
	})
}

type dummyBackend struct {
	domain.Backend
}

func (d *dummyBackend) Cleanup(ctx context.Context, app domain.App) error {
	return nil
}
