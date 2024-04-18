package cleanup_app

import (
	"context"
	"errors"
	"time"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/bus"
	shared "github.com/YuukanOO/seelf/pkg/domain"
)

// Check if the cleanup is needed for a specific app, environment and a specific target.
// It will be skipped if the target is being deleted or if no successful deployment has been made
// in the interval represented by the `From` and `To` parameters.
type Command struct {
	bus.Command[bus.UnitType]

	AppID       string    `json:"app_id"`
	TargetID    string    `json:"target_id"`
	Environment string    `json:"environment"`
	From        time.Time `json:"from"`
	To          time.Time `json:"to"`
}

func (Command) Name_() string        { return "deployment.command.cleanup_app" }
func (c Command) ResourceID() string { return c.AppID }

func Handler(
	reader domain.TargetsReader,
	deploymentsReader domain.DeploymentsReader,
	provider domain.Provider,
) bus.RequestHandler[bus.UnitType, Command] {
	return func(ctx context.Context, cmd Command) (bus.UnitType, error) {
		target, err := reader.GetByID(ctx, domain.TargetID(cmd.TargetID))

		if err != nil {
			if errors.Is(err, apperr.ErrNotFound) {
				return bus.Unit, nil
			}

			return bus.Unit, err
		}

		var (
			appid            = domain.AppID(cmd.AppID)
			env              = domain.Environment(cmd.Environment)
			interval         shared.TimeInterval
			runningOrPending domain.HasRunningOrPendingDeploymentsOnAppTargetEnv
			successful       domain.HasSuccessfulDeploymentsOnAppTargetEnv
		)

		if interval, err = shared.NewTimeInterval(cmd.From, cmd.To); err != nil {
			return bus.Unit, err
		}

		if runningOrPending, successful, err = deploymentsReader.HasDeploymentsOnAppTargetEnv(ctx, appid, target.ID(), env, interval); err != nil {
			return bus.Unit, err
		}

		strategy, err := target.AppCleanupStrategy(runningOrPending, successful)

		if err != nil {
			return bus.Unit, err
		}

		return bus.Unit, provider.Cleanup(ctx, appid, target, env, strategy)
	}
}
