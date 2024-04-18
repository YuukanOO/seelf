package cleanup_target

import (
	"context"
	"errors"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/bus"
)

// Cleanup a target and all its associated resources.
type Command struct {
	bus.Command[bus.UnitType]

	ID string `json:"id"`
}

func (Command) Name_() string        { return "deployment.command.cleanup_target" }
func (c Command) ResourceID() string { return c.ID }

func Handler(
	reader domain.TargetsReader,
	deploymentsReader domain.DeploymentsReader,
	provider domain.Provider,
) bus.RequestHandler[bus.UnitType, Command] {
	return func(ctx context.Context, cmd Command) (bus.UnitType, error) {
		target, err := reader.GetByID(ctx, domain.TargetID(cmd.ID))

		if err != nil {
			// If the target doesn't exist anymore, may be it has been processed by another job in rare case, so just returns
			if errors.Is(err, apperr.ErrNotFound) {
				return bus.Unit, nil
			}

			return bus.Unit, err
		}

		ongoing, err := deploymentsReader.HasRunningOrPendingDeploymentsOnTarget(ctx, target.ID())

		if err != nil {
			return bus.Unit, err
		}

		strategy, err := target.CleanupStrategy(ongoing)

		if err != nil {
			return bus.Unit, err
		}

		return bus.Unit, provider.CleanupTarget(ctx, target, strategy)
	}
}
