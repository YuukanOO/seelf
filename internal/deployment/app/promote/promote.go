package promote

import (
	"context"

	auth "github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/bus"
)

// Promote a deployment to the production environment
type Command struct {
	bus.Command[int]

	AppID            string `json:"-"`
	DeploymentNumber int    `json:"-"`
}

func (Command) Name_() string { return "deployment.command.promote" }

func Handler(
	appsReader domain.AppsReader,
	reader domain.DeploymentsReader,
	writer domain.DeploymentsWriter,
) bus.RequestHandler[int, Command] {
	return func(ctx context.Context, cmd Command) (int, error) {
		app, err := appsReader.GetByID(ctx, domain.AppID(cmd.AppID))

		if err != nil {
			return 0, err
		}

		sourceDeployment, err := reader.GetByID(ctx, domain.DeploymentIDFrom(app.ID(), domain.DeploymentNumber(cmd.DeploymentNumber)))

		if err != nil {
			return 0, err
		}

		number, err := reader.GetNextDeploymentNumber(ctx, app.ID())

		if err != nil {
			return 0, err
		}

		newDeployment, err := app.Promote(sourceDeployment, number, auth.CurrentUser(ctx).MustGet())

		if err != nil {
			return 0, err
		}

		if err := writer.Write(ctx, &newDeployment); err != nil {
			return 0, err
		}

		return int(newDeployment.ID().DeploymentNumber()), nil
	}
}
