package command_test

import (
	"context"
	"testing"

	auth "github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/internal/deployment/app/command"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/deployment/infra/memory"
	"github.com/YuukanOO/seelf/internal/deployment/infra/source/raw"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

func Test_Redeploy(t *testing.T) {
	ctx := auth.WithUserID(context.Background(), "some-uid")
	app := domain.NewApp("my-app", "some-uid")
	appsStore := memory.NewAppsStore(app)

	redeploy := func(existingDeployments ...domain.Deployment) func(ctx context.Context, cmd command.RedeployCommand) (int, error) {
		deploymentsStore := memory.NewDeploymentsStore(existingDeployments...)
		return command.Redeploy(appsStore, deploymentsStore, deploymentsStore)
	}

	t.Run("should fail if application does not exist", func(t *testing.T) {
		uc := redeploy()
		_, err := uc(ctx, command.RedeployCommand{
			AppID: "some-app-id",
		})

		testutil.ErrorIs(t, apperr.ErrNotFound, err)
	})

	t.Run("should fail if source deployment does not exist", func(t *testing.T) {
		uc := redeploy()
		_, err := uc(ctx, command.RedeployCommand{
			AppID:            string(app.ID()),
			DeploymentNumber: 1,
		})

		testutil.ErrorIs(t, apperr.ErrNotFound, err)
	})

	t.Run("should correctly creates a new deployment based on the provided one", func(t *testing.T) {
		dpl, _ := app.NewDeployment(1, raw.Data(""), domain.Production, "some-uid")
		uc := redeploy(dpl)

		number, err := uc(ctx, command.RedeployCommand{
			AppID:            string(dpl.ID().AppID()),
			DeploymentNumber: int(dpl.ID().DeploymentNumber()),
		})

		testutil.IsNil(t, err)
		testutil.Equals(t, 2, number)
	})
}
