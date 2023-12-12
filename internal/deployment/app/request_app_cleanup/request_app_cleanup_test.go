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
	"github.com/YuukanOO/seelf/pkg/testutil"
)

func Test_RequestAppCleanup(t *testing.T) {
	ctx := auth.WithUserID(context.Background(), "some-uid")
	sut := func(existingApps ...*domain.App) bus.RequestHandler[bool, request_app_cleanup.Command] {
		store := memory.NewAppsStore(existingApps...)
		return request_app_cleanup.Handler(store, store)
	}

	t.Run("should fail if the application does not exist", func(t *testing.T) {
		uc := sut()

		success, err := uc(ctx, request_app_cleanup.Command{
			ID: "some-id",
		})

		testutil.ErrorIs(t, apperr.ErrNotFound, err)
		testutil.IsFalse(t, success)
	})

	t.Run("should mark an application has ready for deletion", func(t *testing.T) {
		a := domain.NewApp("my-app", "uid")
		uc := sut(&a)

		success, err := uc(ctx, request_app_cleanup.Command{
			ID: string(a.ID()),
		})

		testutil.IsNil(t, err)
		testutil.IsTrue(t, success)

		testutil.EventIs[domain.AppCleanupRequested](t, &a, 1)
	})
}
