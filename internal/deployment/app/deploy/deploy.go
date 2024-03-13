package deploy

import (
	"context"
	"database/sql/driver"
	"errors"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/storage"
)

// Process a deployment, this is where the magic happen!
type Command struct {
	bus.Command[bus.UnitType]

	AppID            string `json:"app_id"`
	DeploymentNumber int    `json:"deployment_number"`
}

func (Command) Name_() string                  { return "deployment.command.deploy" }
func (c Command) Value() (driver.Value, error) { return storage.ValueJSON(c) }

func init() {
	bus.Marshallable.Register(Command{}, func(s string) (bus.Request, error) { return storage.UnmarshalJSON[Command](s) })
}

func Handler(
	reader domain.DeploymentsReader,
	writer domain.DeploymentsWriter,
	artifactManager domain.ArtifactManager,
	source domain.Source,
	provider domain.Provider,
	targetsReader domain.TargetsReader,
) bus.RequestHandler[bus.UnitType, Command] {
	return func(ctx context.Context, cmd Command) (result bus.UnitType, finalErr error) {
		result = bus.Unit

		depl, err := reader.GetByID(ctx, domain.DeploymentIDFrom(
			domain.AppID(cmd.AppID),
			domain.DeploymentNumber(cmd.DeploymentNumber),
		))

		if err != nil {
			// Deployment does not exist anymore, the app should have been deleted, return early
			if errors.Is(err, apperr.ErrNotFound) {
				return result, nil
			}

			return result, err
		}

		if err = depl.HasStarted(); err != nil {
			// If the deployment could not be started, it probably means the
			// application has been requested for cleanup and the deployment has been
			// cancelled, so the deploy job will never succeed.
			return result, nil
		}

		if err = writer.Write(ctx, &depl); err != nil {
			return result, err
		}

		var (
			target        domain.Target
			deploymentCtx domain.DeploymentContext
			services      domain.Services
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

			if err = depl.HasEnded(services, finalErr); err != nil {
				finalErr = err
				return
			}

			if err = writer.Write(ctx, &depl); err != nil {
				finalErr = err
				return
			}

			finalErr = nil // Don't return any error, the deployment has ended and embed the error if any
		}()

		if target, finalErr = targetsReader.GetByID(ctx, depl.Config().Target()); finalErr != nil {
			return
		}

		// Prepare the build directory
		if deploymentCtx, finalErr = artifactManager.PrepareBuild(ctx, depl); finalErr != nil {
			return
		}

		defer deploymentCtx.Logger().Close()

		// Fetch deployment files
		if finalErr = source.Fetch(ctx, deploymentCtx, depl); finalErr != nil {
			return
		}

		// Ask the provider to actually deploy the app
		if services, finalErr = provider.Run(ctx, deploymentCtx, depl, target); finalErr != nil {
			return
		}

		return
	}
}
