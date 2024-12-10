package cleanup_app

import (
	"context"
	"errors"
	"time"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/bus"
	shared "github.com/YuukanOO/seelf/pkg/domain"
	"github.com/YuukanOO/seelf/pkg/storage"
)

// Check if the cleanup is needed for a specific app, environment and a specific target.
// It will be skipped if the target is being deleted or if no successful deployment has been made
// in the interval represented by the `From` and `To` parameters.
type Command struct {
	bus.AsyncCommand

	AppID       string    `json:"app_id"`
	TargetID    string    `json:"target_id"`
	Environment string    `json:"environment"`
	From        time.Time `json:"from"`
	To          time.Time `json:"to"`
}

func (Command) Name_() string   { return "deployment.command.cleanup_app" }
func (c Command) Group() string { return bus.Group(c.AppID, c.Environment, c.TargetID) }

func Handler(
	reader domain.TargetsReader,
	deploymentsReader domain.DeploymentsReader,
	appsReader domain.AppsReader,
	appsWriter domain.AppsWriter,
	provider domain.Provider,
	uow storage.UnitOfWorkFactory,
) bus.RequestHandler[bus.AsyncResult, Command] {
	return func(ctx context.Context, cmd Command) (result bus.AsyncResult, finalErr error) {
		target, finalErr := reader.GetByID(ctx, domain.TargetID(cmd.TargetID))

		if finalErr != nil {
			if errors.Is(finalErr, apperr.ErrNotFound) {
				finalErr = nil
			}

			return
		}

		var (
			appid            = domain.AppID(cmd.AppID)
			env              = domain.Environment(cmd.Environment)
			interval         shared.TimeInterval
			runningOrPending domain.HasRunningOrPendingDeploymentsOnAppTargetEnv
			successful       domain.HasSuccessfulDeploymentsOnAppTargetEnv
		)

		if interval, finalErr = shared.NewTimeInterval(cmd.From, cmd.To); finalErr != nil {
			return
		}

		if runningOrPending, successful, finalErr = deploymentsReader.HasDeploymentsOnAppTargetEnv(ctx, appid, target.ID(), env, interval); finalErr != nil {
			return
		}

		strategy, finalErr := target.CanAppBeCleaned(runningOrPending, successful)

		if finalErr != nil {
			if errors.Is(finalErr, domain.ErrTargetConfigurationInProgress) ||
				errors.Is(finalErr, domain.ErrRunningOrPendingDeployments) {
				finalErr = nil
				result = bus.AsyncResultDelay
			}

			return
		}

		defer func() {
			if finalErr != nil {
				return
			}

			finalErr = uow.Create(ctx, func(ctx context.Context) error {
				app, err := appsReader.GetByID(ctx, appid)

				if err != nil {
					// Application does not exist anymore, nothing specific to do
					if errors.Is(err, apperr.ErrNotFound) {
						return nil
					}

					return err
				}

				app.CleanedUp(env, target.ID())

				return appsWriter.Write(ctx, &app)
			})
		}()

		finalErr = provider.Cleanup(ctx, appid, target, env, strategy)
		return
	}
}
