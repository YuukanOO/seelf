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
	"github.com/YuukanOO/seelf/pkg/validation"
)

func Test_CreateApp(t *testing.T) {
	ctx := auth.WithUserID(context.Background(), "some-uid")
	create := func(existingApps ...*domain.App) func(context.Context, command.CreateAppCommand) (string, error) {
		store := memory.NewAppsStore(existingApps...)
		return command.CreateApp(store, store)
	}

	t.Run("should require valid inputs", func(t *testing.T) {
		uc := create()
		_, err := uc(ctx, command.CreateAppCommand{})

		testutil.ErrorIs(t, validation.ErrValidationFailed, err)
	})

	t.Run("should fail if the name is already taken", func(t *testing.T) {
		app := domain.NewApp("my-app", "uid")
		uc := create(&app)

		_, err := uc(ctx, command.CreateAppCommand{
			Name: "my-app",
		})

		validationErr, ok := apperr.As[validation.Error](err)
		testutil.IsTrue(t, ok)
		testutil.ErrorIs(t, domain.ErrAppNameAlreadyTaken, validationErr.Fields["name"])
	})

	t.Run("should create a new app if everything is good", func(t *testing.T) {
		uc := create()
		appid, err := uc(ctx, command.CreateAppCommand{
			Name: "my-app",
		})

		testutil.IsNil(t, err)
		testutil.NotEquals(t, "", appid)
	})
}
