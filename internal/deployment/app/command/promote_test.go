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

func Test_Promote(t *testing.T) {
	opts := options{}
	ctx := auth.WithUserID(context.Background(), "some-uid")
	app := domain.NewApp("my-app", "some-uid")
	appsStore := memory.NewAppsStore(app)

	promote := func(existingDeployments ...domain.Deployment) func(ctx context.Context, cmd command.PromoteCommand) (int, error) {
		deploymentsStore := memory.NewDeploymentsStore(existingDeployments...)
		return command.Promote(appsStore, deploymentsStore, deploymentsStore, opts)
	}

	t.Run("should fail if application does not exist", func(t *testing.T) {
		uc := promote()
		_, err := uc(ctx, command.PromoteCommand{
			AppID: "some-app-id",
		})

		testutil.ErrorIs(t, apperr.ErrNotFound, err)
	})

	t.Run("should fail if source deployment does not exist", func(t *testing.T) {
		uc := promote()
		_, err := uc(ctx, command.PromoteCommand{
			AppID:            string(app.ID()),
			DeploymentNumber: 1,
		})

		testutil.ErrorIs(t, apperr.ErrNotFound, err)
	})

	t.Run("should correctly creates a new deployment based on the provided one", func(t *testing.T) {
		dpl, _ := app.NewDeployment(1, meta{}, domain.Staging, opts, "some-uid")
		uc := promote(dpl)

		number, err := uc(ctx, command.PromoteCommand{
			AppID:            string(app.ID()),
			DeploymentNumber: int(dpl.ID().DeploymentNumber()),
		})

		testutil.IsNil(t, err)
		testutil.Equals(t, 2, number)
	})
}
