package get_deployment_log

import (
	"context"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/bus"
)

// Retrieve a deployment log absolute path.
type Query struct {
	bus.Query[string]

	AppID            string `json:"-"`
	DeploymentNumber int    `json:"-"`
}

func (Query) Name_() string { return "deployment.query.get_deployment_log" }

func Handler(
	reader domain.DeploymentsReader,
	artifactManager domain.ArtifactManager,
) bus.RequestHandler[string, Query] {
	return func(ctx context.Context, cmd Query) (string, error) {
		depl, err := reader.GetByID(ctx, domain.DeploymentIDFrom(
			domain.AppID(cmd.AppID),
			domain.DeploymentNumber(cmd.DeploymentNumber),
		))

		if err != nil {
			return "", err
		}

		return artifactManager.LogPath(ctx, depl), nil
	}
}
