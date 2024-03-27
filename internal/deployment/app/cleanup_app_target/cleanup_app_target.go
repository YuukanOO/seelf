package cleanup_app_target

import (
	"context"
	"database/sql/driver"
	"errors"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/storage"
)

type Command struct {
	bus.Command[bus.UnitType]

	AppID       string `json:"id"`
	TargetID    string `json:"target_id"`
	Environment string `json:"environment"`
}

func (Command) Name_() string                  { return "deployment.command.cleanup_app_target" }
func (c Command) Value() (driver.Value, error) { return storage.ValueJSON(c) }

func init() {
	bus.Marshallable.Register(Command{}, func(s string) (bus.Request, error) { return storage.UnmarshalJSON[Command](s) })
}

func Handler(
	reader domain.TargetsReader,
	deploymentsReader domain.DeploymentsReader,
	provider domain.Provider,
) bus.RequestHandler[bus.UnitType, Command] {
	return func(ctx context.Context, cmd Command) (bus.UnitType, error) {
		target, err := reader.GetByID(ctx, domain.TargetID(cmd.TargetID))

		if err != nil {
			if errors.Is(err, apperr.ErrNotFound) {
				return bus.Unit, bus.Ignore(err)
			}

			return bus.Unit, err
		}

		var (
			appid   = domain.AppID(cmd.AppID)
			env     = domain.Environment(cmd.Environment)
			filters domain.GetDeploymentsCountFilters
		)

		filters.Target.Set(target.ID())
		filters.Environment.Set(env)

		count, err := deploymentsReader.GetRunningDeploymentsOnTargetCount(ctx, target.ID(), filters)

		if err != nil {
			return bus.Unit, err
		}

		strategy, err := target.AppCleanupAllowed(count)

		if err != nil {
			return bus.Unit, err
		}

		if strategy == domain.TargetCleanupStrategySkip {
			return bus.Unit, nil
		}

		return bus.Unit, provider.Cleanup(ctx, appid, target, env)
	}
}
