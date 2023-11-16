package command_test

import (
	"context"
	"testing"

	auth "github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/internal/deployment/app/command"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/deployment/infra/memory"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

func Test_RequestAppCleanup(t *testing.T) {
	ctx := auth.WithUserID(context.Background(), "some-uid")
	delete := func(existingApps ...*domain.App) func(context.Context, command.RequestAppCleanupCommand) error {
		store := memory.NewAppsStore(existingApps...)
		return command.RequestAppCleanup(store, store)
	}

	t.Run("should fail if the application does not exist", func(t *testing.T) {
		uc := delete()

		err := uc(ctx, command.RequestAppCleanupCommand{
			ID: "some-id",
		})

		testutil.ErrorIs(t, apperr.ErrNotFound, err)
	})

	t.Run("should mark an application has ready for deletion", func(t *testing.T) {
		app := domain.NewApp("my-app", "uid")
		uc := delete(&app)

		err := uc(ctx, command.RequestAppCleanupCommand{
			ID: string(app.ID()),
		})

		testutil.IsNil(t, err)
		testutil.EventIs[domain.AppCleanupRequested](t, &app, 1)
	})
}
