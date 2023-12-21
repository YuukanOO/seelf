package promote_test

import (
	"context"
	"testing"

	auth "github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/internal/deployment/app/promote"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/deployment/infra/memory"
	"github.com/YuukanOO/seelf/internal/deployment/infra/source/raw"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

func Test_Promote(t *testing.T) {
	ctx := auth.WithUserID(context.Background(), "some-uid")
	a := domain.NewApp("my-app", "some-uid")
	appsStore := memory.NewAppsStore(&a)

	sut := func(existingDeployments ...*domain.Deployment) bus.RequestHandler[int, promote.Command] {
		deploymentsStore := memory.NewDeploymentsStore(existingDeployments...)
		return promote.Handler(appsStore, deploymentsStore, deploymentsStore)
	}

	t.Run("should fail if application does not exist", func(t *testing.T) {
		uc := sut()
		num, err := uc(ctx, promote.Command{
			AppID: "some-app-id",
		})

		testutil.ErrorIs(t, apperr.ErrNotFound, err)
		testutil.Equals(t, 0, num)
	})

	t.Run("should fail if source deployment does not exist", func(t *testing.T) {
		uc := sut()
		num, err := uc(ctx, promote.Command{
			AppID:            string(a.ID()),
			DeploymentNumber: 1,
		})

		testutil.ErrorIs(t, apperr.ErrNotFound, err)
		testutil.Equals(t, 0, num)
	})

	t.Run("should correctly creates a new deployment based on the provided one", func(t *testing.T) {
		dpl, _ := a.NewDeployment(1, raw.Data(""), domain.Staging, "some-uid")
		uc := sut(&dpl)

		number, err := uc(ctx, promote.Command{
			AppID:            string(dpl.ID().AppID()),
			DeploymentNumber: int(dpl.ID().DeploymentNumber()),
		})

		testutil.IsNil(t, err)
		testutil.Equals(t, 2, number)
	})
}
