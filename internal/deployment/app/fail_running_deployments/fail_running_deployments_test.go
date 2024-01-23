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
	"github.com/YuukanOO/seelf/pkg/must"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

func Test_FailRunningDeployments(t *testing.T) {
	ctx := auth.WithUserID(context.Background(), "some-uid")
	app := must.Panic(domain.NewApp("my-app", domain.NewEnvironmentConfig("1"), domain.NewEnvironmentConfig("1"), "some-uid", domain.AppNamingAvailable))

	sut := func(existingDeployments ...*domain.Deployment) bus.RequestHandler[bus.UnitType, fail_running_deployments.Command] {
		deploymentsStore := memory.NewDeploymentsStore(existingDeployments...)
		return fail_running_deployments.Handler(deploymentsStore, deploymentsStore)
	}

	t.Run("should reset running deployments", func(t *testing.T) {
		errReset := errors.New("server_reset")

		started := must.Panic(app.NewDeployment(2, raw.Data(""), domain.Production, "some-uid"))
		testutil.IsNil(t, started.HasStarted())

		succeeded := must.Panic(app.NewDeployment(1, raw.Data(""), domain.Production, "some-uid"))
		testutil.IsNil(t, succeeded.HasStarted())
		testutil.IsNil(t, succeeded.HasEnded(domain.Services{}, nil))

		uc := sut(&started, &succeeded)

		r, err := uc(ctx, fail_running_deployments.Command{
			Reason: errReset,
		})

		testutil.IsNil(t, err)
		testutil.Equals(t, bus.Unit, r)

		evt := testutil.EventIs[domain.DeploymentStateChanged](t, &started, 2)
		testutil.Equals(t, domain.DeploymentStatusFailed, evt.State.Status())
		testutil.Equals(t, errReset.Error(), evt.State.ErrCode().MustGet())

		evt = testutil.EventIs[domain.DeploymentStateChanged](t, &succeeded, 2)
		testutil.Equals(t, domain.DeploymentStatusSucceeded, evt.State.Status())
	})
}
