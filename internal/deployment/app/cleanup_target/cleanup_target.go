package cleanup_target

import (
	"context"
	"database/sql/driver"
	"errors"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/storage"
)

// Cleanup a target and all its associated resources.
type Command struct {
	bus.Command[bus.UnitType]

	ID string `json:"id"`
}

func (Command) Name_() string                  { return "deployment.command.cleanup_target" }
func (c Command) Value() (driver.Value, error) { return storage.ValueJSON(c) }

func init() {
	bus.Marshallable.Register(Command{}, func(s string) (bus.Request, error) { return storage.UnmarshalJSON[Command](s) })
}

func Handler(
	reader domain.TargetsReader,
	writer domain.TargetsWriter,
	deploymentsReader domain.DeploymentsReader,
	provider domain.Provider,
) bus.RequestHandler[bus.UnitType, Command] {
	return func(ctx context.Context, cmd Command) (bus.UnitType, error) {
		target, err := reader.GetByID(ctx, domain.TargetID(cmd.ID))

		if err != nil {
			// If the target doesn't exist anymore, may be it has been processed by another job in rare case, so just returns
			if errors.Is(err, apperr.ErrNotFound) {
				return bus.Unit, bus.Ignore(err)
			}

			return bus.Unit, err
		}

		count, err := deploymentsReader.GetRunningDeploymentsOnTargetCount(ctx, target.ID())

		if err != nil {
			return bus.Unit, err
		}

		strategy, err := target.Delete(count)

		if err != nil {
			return bus.Unit, err
		}

		// Don't bother with the provider if the strategy is to skip the cleanup
		if strategy == domain.TargetCleanupStrategySkip {
			return bus.Unit, writer.Write(ctx, &target)
		}

		if err = provider.CleanupTarget(ctx, target, strategy); err != nil {
			return bus.Unit, err
		}

		return bus.Unit, writer.Write(ctx, &target)
	}
}
