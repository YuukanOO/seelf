package deploy

import (
	"context"
	"errors"
	"strconv"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/bus"
)

// Process a deployment, this is where the magic happen!
type Command struct {
	bus.AsyncCommand

	AppID            string `json:"app_id"`
	DeploymentNumber int    `json:"deployment_number"`
	Environment      string `json:"environment"`
	TargetID         string `json:"target_id"`
}

func (Command) Name_() string        { return "deployment.command.deploy" }
func (c Command) ResourceID() string { return c.AppID + "-" + strconv.Itoa(c.DeploymentNumber) }
func (c Command) Group() string      { return bus.Group(c.AppID, c.Environment, c.TargetID) }

// Handle the deployment process.
// If an unexpected error occurs during this process, it uses the bus.PreserveOrder function
// to make sure all deployments are processed linearly.
func Handler(
	reader domain.DeploymentsReader,
	writer domain.DeploymentsWriter,
	artifactManager domain.ArtifactManager,
	source domain.Source,
	provider domain.Provider,
	targetsReader domain.TargetsReader,
	registriesReader domain.RegistriesReader,
) bus.RequestHandler[bus.AsyncResult, Command] {
	return func(ctx context.Context, cmd Command) (result bus.AsyncResult, finalErr error) {
		deployment, err := reader.GetByID(ctx, domain.DeploymentIDFrom(
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

		// If the target does not exist, fail the deployment
		// If the target is not ready, returns early without starting the deployment
		target, targetErr := targetsReader.GetByID(ctx, deployment.Config().Target())

		if targetErr != nil && !errors.Is(targetErr, apperr.ErrNotFound) {
			return result, targetErr
		}

		if err = deployment.HasStarted(); err != nil {
			// If the deployment could not be started, it probably means the
			// application has been requested for cleanup and the deployment has been
			// cancelled, so the deploy job will never succeed.
			return result, nil
		}

		var targetAvailabilityErr error

		if targetErr == nil {
			targetAvailabilityErr = target.CheckAvailability()

			// Target configuration is in progress, just retry the job later without writing
			// the deployment, keep it in pending state
			if errors.Is(targetAvailabilityErr, domain.ErrTargetConfigurationInProgress) {
				return bus.AsyncResultDelay, nil
			}
		}

		if err = writer.Write(ctx, &deployment); err != nil {
			return result, err
		}

		var (
			deploymentCtx domain.DeploymentContext
			services      domain.Services
			registries    []domain.Registry
		)

		// This one is a special case to avoid to avoid many branches
		// checking for errors when writing the domain.
		// Based on wether or not there was an error, it will update the deployment
		// accordingly.
		//
		// Unlike other async jobs, the unit of work is not needed here because the deployment could not be
		// updated by someone else. And if it was, a concurrent update error will be returned.
		defer func() {
			// Since the deployment process could take some time, retrieve a fresh version of the
			// deployment right now
			if deployment, err = reader.GetByID(ctx, deployment.ID()); err != nil {
				if errors.Is(err, apperr.ErrNotFound) {
					err = nil
				}

				finalErr = err
				return
			}

			// An error means it has already been handled
			if err = deployment.HasEnded(services, finalErr); err != nil {
				finalErr = nil
				return
			}

			finalErr = writer.Write(ctx, &deployment)
		}()

		// Prepare the build directory
		if deploymentCtx, finalErr = artifactManager.PrepareBuild(ctx, deployment); finalErr != nil {
			return
		}

		defer deploymentCtx.Logger().Close()

		// If the target does not exist, let's fail the deployment correctly
		if finalErr = targetErr; finalErr != nil {
			return
		}

		// Target not available, fail the deployment
		if finalErr = targetAvailabilityErr; finalErr != nil {
			return
		}

		// Fetch deployment files
		if finalErr = source.Fetch(ctx, deploymentCtx, deployment); finalErr != nil {
			return
		}

		// Fetch custom registries
		if registries, finalErr = registriesReader.GetAll(ctx); finalErr != nil {
			return
		}

		// Ask the provider to actually deploy the app
		if services, finalErr = provider.Deploy(ctx, deploymentCtx, deployment, target, registries); finalErr != nil {
			return
		}

		return
	}
}
