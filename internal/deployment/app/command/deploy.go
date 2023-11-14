package command

import (
	"context"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
)

type DeployCommand struct {
	AppID            string
	DeploymentNumber int
}

// Process a deployment, this is where the magic happen!
func Deploy(
	reader domain.DeploymentsReader,
	writer domain.DeploymentsWriter,
	artifactManager domain.ArtifactManager,
	source domain.Source,
	backend domain.Backend,
) func(context.Context, DeployCommand) error {
	return func(ctx context.Context, cmd DeployCommand) (finalErr error) {
		depl, err := reader.GetByID(ctx, domain.DeploymentIDFrom(
			domain.AppID(cmd.AppID),
			domain.DeploymentNumber(cmd.DeploymentNumber),
		))

		if err != nil {
			return err
		}

		err = depl.HasStarted()

		if err != nil {
			return err
		}

		if err = writer.Write(ctx, &depl); err != nil {
			return err
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

		return nil
	}
}
