package command_test

import (
	"context"
	"testing"

	"github.com/YuukanOO/seelf/internal/deployment/app/command"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/deployment/infra/memory"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

type initialData struct {
	existingApps        []domain.App
	existingDeployments []domain.Deployment
}

func Test_CleanupApp(t *testing.T) {
	ctx := context.Background()
	cleanup := func(initialData initialData) func(context.Context, command.CleanupAppCommand) error {
		store := memory.NewAppsStore(initialData.existingApps...)
		deploymentsStore := memory.NewDeploymentsStore(initialData.existingDeployments...)
		return command.CleanupApp(deploymentsStore, store, store, &dummyBackend{})
	}

	t.Run("should fail if the application does not exist", func(t *testing.T) {
		uc := cleanup(initialData{})

		err := uc(ctx, command.CleanupAppCommand{
			ID: "some-id",
		})

		testutil.ErrorIs(t, apperr.ErrNotFound, err)
	})

	t.Run("should fail if the application cleanup as not been requested", func(t *testing.T) {
		app := domain.NewApp("my-app", "uid")
		uc := cleanup(initialData{
			existingApps: []domain.App{app},
		})

		err := uc(ctx, command.CleanupAppCommand{
			ID: string(app.ID()),
		})

		testutil.ErrorIs(t, domain.ErrAppCleanupNeeded, err)
	})

	t.Run("should fail if there are still pending or running deployments", func(t *testing.T) {
		app := domain.NewApp("my-app", "uid")
		depl, _ := app.NewDeployment(1, domain.NewMeta("some", "data"), domain.Production, options{}, "uid")
		app.RequestCleanup("uid")

		uc := cleanup(initialData{
			existingApps:        []domain.App{app},
			existingDeployments: []domain.Deployment{depl},
		})

		err := uc(ctx, command.CleanupAppCommand{
			ID: string(app.ID()),
		})

		testutil.ErrorIs(t, domain.ErrAppHasRunningOrPendingDeployments, err)
	})

	t.Run("should succeed if everything is good", func(t *testing.T) {
		app := domain.NewApp("my-app", "uid")
		app.RequestCleanup("uid")

		uc := cleanup(initialData{
			existingApps: []domain.App{app},
		})

		err := uc(ctx, command.CleanupAppCommand{
			ID: string(app.ID()),
		})

		testutil.IsNil(t, err)
	})
}
