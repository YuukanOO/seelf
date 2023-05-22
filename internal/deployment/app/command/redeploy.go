package command

import (
	"context"

	auth "github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
)

type RedeployCommand struct {
	AppID            string `json:"-"`
	DeploymentNumber int    `json:"-"`
}

// Request a redeploy of the given deployment.
func Redeploy(
	appsReader domain.AppsReader,
	reader domain.DeploymentsReader,
	writer domain.DeploymentsWriter,
	deploymentDirTemplate domain.DeploymentDirTemplate,
) func(context.Context, RedeployCommand) (int, error) {
	return func(ctx context.Context, cmd RedeployCommand) (int, error) {
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

		newDeployment, err := app.Redeploy(sourceDeployment, number, deploymentDirTemplate, auth.CurrentUser(ctx).MustGet())

		if err != nil {
			return 0, err
		}

		if err := writer.Write(ctx, &newDeployment); err != nil {
			return 0, err
		}

		return int(newDeployment.ID().DeploymentNumber()), nil
	}
}
