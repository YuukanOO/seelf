package queue_deployment

import (
	"context"

	auth "github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/validate"
)

// Queue a deployment for a given app and source. It will returns the deployment number
// created.
type Command struct {
	bus.Command[int]

	AppID       string `json:"-"`
	Environment string `json:"environment" form:"environment"`
	Source      any    `json:"-"`
}

func (Command) Name_() string { return "deployment.command.queue_deployment" }

func Handler(
	appsReader domain.AppsReader,
	reader domain.DeploymentsReader,
	writer domain.DeploymentsWriter,
	source domain.Source,
) bus.RequestHandler[int, Command] {
	return func(ctx context.Context, cmd Command) (int, error) {
		var env domain.EnvironmentName

		if err := validate.Struct(validate.Of{
			"environment": validate.Value(cmd.Environment, &env, domain.EnvironmentNameFrom),
		}); err != nil {
			return 0, err
		}

		app, err := appsReader.GetByID(ctx, domain.AppID(cmd.AppID))

		if err != nil {
			return 0, err
		}

		meta, err := source.Prepare(ctx, app, cmd.Source)

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
