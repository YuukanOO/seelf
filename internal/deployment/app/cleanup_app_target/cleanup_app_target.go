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

		beenReadyAtLeastOnce, err := target.CheckAvailability()

		// Target configuration has failed but the target has never been reachable so no need
		// to cleanup anything or the target is being deleted, everything will be removed, no need to
		// case about it
		if (errors.Is(err, domain.ErrTargetConfigurationFailed) && !beenReadyAtLeastOnce) ||
			errors.Is(err, domain.ErrTargetDeleteRequested) {
			return bus.Unit, nil
		}

		if err != nil {
			return bus.Unit, err
		}

		return bus.Unit, provider.Cleanup(ctx, domain.AppID(cmd.AppID), target, domain.Environment(cmd.Environment))
	}
}
