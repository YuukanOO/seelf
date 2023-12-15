package deploy

import (
	"context"
	"fmt"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/monad"
)

// Upon receiving a deployment created event, queue a job to deploy the application.
func OnDeploymentCreatedHandler(scheduler bus.Scheduler) bus.SignalHandler[domain.DeploymentCreated] {
	return func(ctx context.Context, evt domain.DeploymentCreated) error {
		cmd := Command{
			AppID:            string(evt.ID.AppID()),
			DeploymentNumber: int(evt.ID.DeploymentNumber()),
		}

		// Only one deployment per app per environment can be processed at a given time.
		dedupeName := monad.Value(fmt.Sprintf("%s.%s", cmd.Name_(), evt.Config.ProjectName()))

		return scheduler.Queue(ctx, cmd, dedupeName, bus.JobErrPolicyIgnore)
	}
}

func init() {
	bus.RegisterForMarshalling[Command]()
}
