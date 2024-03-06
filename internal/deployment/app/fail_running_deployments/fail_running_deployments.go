package fail_running_deployments

import (
	"context"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/bus"
)

// Mark all running deployments as failed with the given reason. This is mostly used
// when the server has crashed or has been hard resetted and some job does not ended
// correctly. They will need a redeploy.
type Command struct {
	bus.Command[bus.UnitType]

	Reason error `json:"-"`
}

func (Command) Name_() string { return "deployment.command.fail_running_deployments" }

func Handler(writer domain.DeploymentsWriter) bus.RequestHandler[bus.UnitType, Command] {
	return func(ctx context.Context, cmd Command) (bus.UnitType, error) {
		return bus.Unit, writer.FailDeployments(ctx, domain.DeploymentStatusRunning, cmd.Reason)
	}
}
