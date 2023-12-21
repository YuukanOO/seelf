package fail_running_deployments

import (
	"context"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/collections"
)

// Mark all running deployments as failed with the given reason. This is mostly used
// when the server has crashed or has been hard resetted and some job does not ended
// correctly. They will need a redeploy.
type Command struct {
	bus.Command[bus.UnitType]

	Reason error `json:"-"`
}

func (Command) Name_() string { return "deployment.command.fail_running_deployments" }

func Handler(
	reader domain.DeploymentsReader,
	writer domain.DeploymentsWriter,
) bus.RequestHandler[bus.UnitType, Command] {
	return func(ctx context.Context, cmd Command) (bus.UnitType, error) {
		deployments, err := reader.GetRunningDeployments(ctx)

		if err != nil {
			return bus.Unit, err
		}

		for idx := range deployments {
			err = deployments[idx].HasEnded(nil, cmd.Reason)

			if err != nil {
				return bus.Unit, err
			}
		}

		if err = writer.Write(ctx, collections.ToPointers(deployments)...); err != nil {
			return bus.Unit, err
		}

		return bus.Unit, nil
	}
}
