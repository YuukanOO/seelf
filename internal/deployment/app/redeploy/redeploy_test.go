package redeploy_test

import (
	"context"
	"testing"

	auth "github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/internal/deployment/app/redeploy"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/deployment/infra/memory"
	"github.com/YuukanOO/seelf/internal/deployment/infra/source/raw"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/must"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

func Test_Redeploy(t *testing.T) {
	ctx := auth.WithUserID(context.Background(), "some-uid")
	app := must.Panic(domain.NewApp("my-app",
		domain.NewEnvironmentConfigRequirement(domain.NewEnvironmentConfig("1"), true, true),
		domain.NewEnvironmentConfigRequirement(domain.NewEnvironmentConfig("1"), true, true), "some-uid"))
	appsStore := memory.NewAppsStore(&app)

	sut := func(existingDeployments ...*domain.Deployment) bus.RequestHandler[int, redeploy.Command] {
		deploymentsStore := memory.NewDeploymentsStore(existingDeployments...)
		return redeploy.Handler(appsStore, deploymentsStore, deploymentsStore)
	}

	t.Run("should fail if application does not exist", func(t *testing.T) {
		uc := sut()
		num, err := uc(ctx, redeploy.Command{
			AppID: "some-app-id",
		})

		testutil.ErrorIs(t, apperr.ErrNotFound, err)
		testutil.Equals(t, 0, num)
	})

	t.Run("should fail if source deployment does not exist", func(t *testing.T) {
		uc := sut()
		num, err := uc(ctx, redeploy.Command{
			AppID:            string(app.ID()),
			DeploymentNumber: 1,
		})

		testutil.ErrorIs(t, apperr.ErrNotFound, err)
		testutil.Equals(t, 0, num)

	})

	t.Run("should correctly creates a new deployment based on the provided one", func(t *testing.T) {
		dpl, _ := app.NewDeployment(1, raw.Data(""), domain.Production, "some-uid")
		uc := sut(&dpl)

		num, err := uc(ctx, redeploy.Command{
			AppID:            string(dpl.ID().AppID()),
			DeploymentNumber: int(dpl.ID().DeploymentNumber()),
		})

		testutil.IsNil(t, err)
		testutil.Equals(t, 2, num)
	})
}
