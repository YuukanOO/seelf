package cleanup_app_test

import (
	"context"
	"os"
	"testing"

	"github.com/YuukanOO/seelf/cmd"
	"github.com/YuukanOO/seelf/internal/deployment/app/cleanup_app"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/deployment/infra"
	"github.com/YuukanOO/seelf/internal/deployment/infra/memory"
	"github.com/YuukanOO/seelf/internal/deployment/infra/source/raw"
	"github.com/YuukanOO/seelf/pkg/apperr"
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

	sut := func(initialData initialData) bus.RequestHandler[bool, cleanup_app.Command] {
		opts := cmd.DefaultConfiguration(cmd.WithTestDefaults())
		appsStore := memory.NewAppsStore(initialData.existingApps...)
		deploymentsStore := memory.NewDeploymentsStore(initialData.existingDeployments...)
		artifactManager := infra.NewLocalArtifactManager(opts, logger)

		t.Cleanup(func() {
			os.RemoveAll(opts.DataDir())
		})

		return cleanup_app.Handler(deploymentsStore, appsStore, appsStore, artifactManager, &dummyBackend{})
	}

	t.Run("should fail if the application does not exist", func(t *testing.T) {
		uc := sut(initialData{})

		success, err := uc(ctx, cleanup_app.Command{
			ID: "some-id",
		})

		testutil.ErrorIs(t, apperr.ErrNotFound, err)
		testutil.IsFalse(t, success)
	})

	t.Run("should fail if the application cleanup as not been requested", func(t *testing.T) {
		a := domain.NewApp("my-app", "uid")
		uc := sut(initialData{
			existingApps: []*domain.App{&a},
		})

		success, err := uc(ctx, cleanup_app.Command{
			ID: string(a.ID()),
		})

		testutil.ErrorIs(t, domain.ErrAppCleanupNeeded, err)
		testutil.IsFalse(t, success)
	})

	t.Run("should fail if there are still pending or running deployments", func(t *testing.T) {
		a := domain.NewApp("my-app", "uid")
		depl, _ := a.NewDeployment(1, raw.Data(""), domain.Production, "uid")
		a.RequestCleanup("uid")

		uc := sut(initialData{
			existingApps:        []*domain.App{&a},
			existingDeployments: []*domain.Deployment{&depl},
		})

		success, err := uc(ctx, cleanup_app.Command{
			ID: string(a.ID()),
		})

		testutil.ErrorIs(t, domain.ErrAppHasRunningOrPendingDeployments, err)
		testutil.IsFalse(t, success)
	})

	t.Run("should succeed if everything is good", func(t *testing.T) {
		a := domain.NewApp("my-app", "uid")
		a.RequestCleanup("uid")

		uc := sut(initialData{
			existingApps: []*domain.App{&a},
		})

		success, err := uc(ctx, cleanup_app.Command{
			ID: string(a.ID()),
		})

		testutil.IsNil(t, err)
		testutil.IsTrue(t, success)
		testutil.EventIs[domain.AppDeleted](t, &a, 2)
	})
}

type dummyBackend struct {
	domain.Backend
}

func (d *dummyBackend) Cleanup(ctx context.Context, app domain.App) error {
	return nil
}
