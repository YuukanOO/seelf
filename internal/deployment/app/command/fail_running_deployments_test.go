package command_test

import (
	"context"
	"errors"
	"testing"

	auth "github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/internal/deployment/app/command"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/deployment/infra/memory"
	"github.com/YuukanOO/seelf/internal/deployment/infra/source/raw"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

func Test_FailRunningDeployments(t *testing.T) {
	ctx := auth.WithUserID(context.Background(), "some-uid")
	app := domain.NewApp("my-app", "some-uid")

	fail := func(existingDeployments ...domain.Deployment) (func(context.Context, error) error, domain.DeploymentsReader) {
		deploymentsStore := memory.NewDeploymentsStore(existingDeployments...)
		return command.FailRunningDeployments(deploymentsStore, deploymentsStore), deploymentsStore
	}

	t.Run("should reset running deployments", func(t *testing.T) {
		errReset := errors.New("server_reset")

		started, _ := app.NewDeployment(2, raw.Data(""), domain.Production, "some-uid")
		err := started.HasStarted()

		testutil.IsNil(t, err)

		succeeded, _ := app.NewDeployment(1, raw.Data(""), domain.Production, "some-uid")
		succeeded.HasStarted()
		err = succeeded.HasEnded(domain.Services{}, nil)

		testutil.IsNil(t, err)

		uc, store := fail(started, succeeded)

		err = uc(ctx, errReset)

		testutil.IsNil(t, err)

		started, _ = store.GetByID(ctx, started.ID())
		evt := testutil.EventIs[domain.DeploymentStateChanged](t, &started, 2)
		testutil.Equals(t, domain.DeploymentStatusFailed, evt.State.Status())
		testutil.Equals(t, errReset.Error(), evt.State.ErrCode().MustGet())

		succeeded, _ = store.GetByID(ctx, succeeded.ID())
		evt = testutil.EventIs[domain.DeploymentStateChanged](t, &succeeded, 2)
		testutil.Equals(t, domain.DeploymentStatusSucceeded, evt.State.Status())
	})
}
