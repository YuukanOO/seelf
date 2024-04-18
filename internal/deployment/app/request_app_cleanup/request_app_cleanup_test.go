package request_app_cleanup_test

import (
	"context"
	"testing"

	auth "github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/internal/deployment/app/request_app_cleanup"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/deployment/infra/memory"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/must"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

func Test_RequestAppCleanup(t *testing.T) {
	ctx := auth.WithUserID(context.Background(), "some-uid")
	sut := func(existingApps ...*domain.App) bus.RequestHandler[bus.UnitType, request_app_cleanup.Command] {
		store := memory.NewAppsStore(existingApps...)
		return request_app_cleanup.Handler(store, store)
	}

	t.Run("should fail if the application does not exist", func(t *testing.T) {
		uc := sut()

		r, err := uc(ctx, request_app_cleanup.Command{
			ID: "some-id",
		})

		testutil.ErrorIs(t, apperr.ErrNotFound, err)
		testutil.Equals(t, bus.Unit, r)
	})

	t.Run("should mark an application has ready for deletion", func(t *testing.T) {
		app := must.Panic(domain.NewApp("my-app",
			domain.NewEnvironmentConfigRequirement(domain.NewEnvironmentConfig("1"), true, true),
			domain.NewEnvironmentConfigRequirement(domain.NewEnvironmentConfig("1"), true, true), "some-uid"))
		uc := sut(&app)

		r, err := uc(ctx, request_app_cleanup.Command{
			ID: string(app.ID()),
		})

		testutil.IsNil(t, err)
		testutil.Equals(t, bus.Unit, r)

		testutil.EventIs[domain.AppCleanupRequested](t, &app, 1)
	})
}
