package create_app_test

import (
	"context"
	"testing"

	auth "github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/internal/deployment/app/create_app"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/deployment/infra/memory"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/testutil"
	"github.com/YuukanOO/seelf/pkg/validation"
)

func Test_CreateApp(t *testing.T) {
	ctx := auth.WithUserID(context.Background(), "some-uid")
	sut := func(existingApps ...*domain.App) bus.RequestHandler[string, create_app.Command] {
		store := memory.NewAppsStore(existingApps...)
		return create_app.Handler(store, store)
	}

	t.Run("should require valid inputs", func(t *testing.T) {
		uc := sut()
		id, err := uc(ctx, create_app.Command{})

		testutil.ErrorIs(t, validation.ErrValidationFailed, err)
		testutil.Equals(t, "", id)
	})

	t.Run("should fail if the name is already taken", func(t *testing.T) {
		a := domain.NewApp("my-app", "uid")
		uc := sut(&a)

		id, err := uc(ctx, create_app.Command{
			Name: "my-app",
		})

		validationErr, ok := apperr.As[validation.Error](err)
		testutil.IsTrue(t, ok)
		testutil.Equals(t, "", id)
		testutil.ErrorIs(t, domain.ErrAppNameAlreadyTaken, validationErr.Fields["name"])
	})

	t.Run("should create a new app if everything is good", func(t *testing.T) {
		uc := sut()
		id, err := uc(ctx, create_app.Command{
			Name: "my-app",
		})

		testutil.IsNil(t, err)
		testutil.NotEquals(t, "", id)
	})
}
