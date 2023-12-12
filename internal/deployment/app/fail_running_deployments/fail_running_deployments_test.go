package fail_running_deployments_test

import (
	"context"
	"errors"
	"testing"

	auth "github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/internal/deployment/app/fail_running_deployments"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/deployment/infra/memory"
	"github.com/YuukanOO/seelf/internal/deployment/infra/source/raw"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

func Test_FailRunningDeployments(t *testing.T) {
	ctx := auth.WithUserID(context.Background(), "some-uid")
	a := domain.NewApp("my-app", "some-uid")

	sut := func(existingDeployments ...*domain.Deployment) bus.RequestHandler[bool, fail_running_deployments.Command] {
		deploymentsStore := memory.NewDeploymentsStore(existingDeployments...)
		return fail_running_deployments.Handler(deploymentsStore, deploymentsStore)
	}

	t.Run("should reset running deployments", func(t *testing.T) {
		errReset := errors.New("server_reset")

		started, _ := a.NewDeployment(2, raw.Data(""), domain.Production, "some-uid")
		err := started.HasStarted()

		testutil.IsNil(t, err)

		succeeded, _ := a.NewDeployment(1, raw.Data(""), domain.Production, "some-uid")
		succeeded.HasStarted()
		err = succeeded.HasEnded(domain.Services{}, nil)

		testutil.IsNil(t, err)

		uc := sut(&started, &succeeded)

		success, err := uc(ctx, fail_running_deployments.Command{
			Reason: errReset,
		})

		testutil.IsNil(t, err)
		testutil.IsTrue(t, success)

		evt := testutil.EventIs[domain.DeploymentStateChanged](t, &started, 2)
		testutil.Equals(t, domain.DeploymentStatusFailed, evt.State.Status())
		testutil.Equals(t, errReset.Error(), evt.State.ErrCode().MustGet())

		evt = testutil.EventIs[domain.DeploymentStateChanged](t, &succeeded, 2)
		testutil.Equals(t, domain.DeploymentStatusSucceeded, evt.State.Status())
	})
}
