package command

import (
	"context"

	auth "github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/validation"
)

type QueueDeploymentCommand struct {
	AppID       string `json:"-"`
	Environment string `json:"-"`
	Payload     any    `json:"-"`
}

// Queue a deployment for a given app and source. It will returns the deployment number
// created.
func QueueDeployment(
	appsReader domain.AppsReader,
	reader domain.DeploymentsReader,
	writer domain.DeploymentsWriter,
	source domain.Source,
) func(ctx context.Context, cmd QueueDeploymentCommand) (int, error) {
	return func(ctx context.Context, cmd QueueDeploymentCommand) (int, error) {
		var env domain.Environment

		if err := validation.Check(validation.Of{
			"environment": validation.Value(cmd.Environment, &env, domain.EnvironmentFrom),
		}); err != nil {
			return 0, err
		}

		app, err := appsReader.GetByID(ctx, domain.AppID(cmd.AppID))

		if err != nil {
			return 0, err
		}

		meta, err := source.Prepare(app, cmd.Payload)

		if err != nil {
			return 0, err
		}

		number, err := reader.GetNextDeploymentNumber(ctx, app.ID())

		if err != nil {
			return 0, err
		}

		dpl, err := app.NewDeployment(number, meta, env, auth.CurrentUser(ctx).MustGet())

		if err != nil {
			return 0, err
		}

		if err := writer.Write(ctx, &dpl); err != nil {
			return 0, err
		}

		return int(dpl.ID().DeploymentNumber()), nil
	}
}
