package deploy

import (
	"context"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/bus"
)

// Process a deployment, this is where the magic happen!
type Command struct {
	bus.Command[bool]

	AppID            string `json:"-"`
	DeploymentNumber int    `json:"-"`
}

func (Command) Name_() string { return "deployment.command.deploy" }

func Handler(
	reader domain.DeploymentsReader,
	writer domain.DeploymentsWriter,
	artifactManager domain.ArtifactManager,
	source domain.Source,
	backend domain.Backend,
) bus.RequestHandler[bool, Command] {
	return func(ctx context.Context, cmd Command) (success bool, finalErr error) {
		depl, err := reader.GetByID(ctx, domain.DeploymentIDFrom(
			domain.AppID(cmd.AppID),
			domain.DeploymentNumber(cmd.DeploymentNumber),
		))

		if err != nil {
			return success, err
		}

		err = depl.HasStarted()

		if err != nil {
			return success, err
		}

		if err = writer.Write(ctx, &depl); err != nil {
			return success, err
		}

		var (
			buildDirectory string
			logger         domain.DeploymentLogger
			services       domain.Services
		)

		// This one is a special case to avoid to avoid many branches
		// checking for errors when writing the domain.
		// Based on wether or not there was an error, it will update the deployment
		// accordingly.
		defer func() {
			// Since the deployment process could take some time, retrieve a fresh version of the
			// deployment right now
			if depl, err = reader.GetByID(ctx, depl.ID()); err != nil {
				finalErr = err
				return
			}

			stateErr := depl.HasEnded(services, finalErr)

			if stateErr != nil {
				finalErr = stateErr
				return
			}

			if werr := writer.Write(ctx, &depl); werr != nil {
				finalErr = werr
				return
			}
		}()

		// Prepare the build directory
		if buildDirectory, logger, finalErr = artifactManager.PrepareBuild(ctx, depl); finalErr != nil {
			return
		}

		defer logger.Close()

		// Fetch deployment files
		if finalErr = source.Fetch(ctx, buildDirectory, logger, depl); finalErr != nil {
			return
		}

		// Ask the backend to actually deploy the app
		if services, finalErr = backend.Run(ctx, buildDirectory, logger, depl); finalErr != nil {
			return
		}

		return true, nil
	}
}
