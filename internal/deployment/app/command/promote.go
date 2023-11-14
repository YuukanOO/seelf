package command

import (
	"context"

	auth "github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
)

type PromoteCommand struct {
	AppID            string `json:"-"`
	DeploymentNumber int    `json:"-"`
}

// Promote a deployment to the production environment
func Promote(
	appsReader domain.AppsReader,
	reader domain.DeploymentsReader,
	writer domain.DeploymentsWriter,
) func(context.Context, PromoteCommand) (int, error) {
	return func(ctx context.Context, cmd PromoteCommand) (int, error) {
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
