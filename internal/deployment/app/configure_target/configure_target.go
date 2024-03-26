package configure_target

import (
	"context"
	"database/sql/driver"
	"errors"
	"time"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/storage"
)

type Command struct {
	bus.Command[bus.UnitType]

	ID      string    `json:"id"`
	Version time.Time `json:"version"`
}

func (Command) Name_() string                  { return "deployment.command.configure_target" }
func (c Command) Value() (driver.Value, error) { return storage.ValueJSON(c) }

func init() {
	bus.Marshallable.Register(Command{}, func(s string) (bus.Request, error) { return storage.UnmarshalJSON[Command](s) })
}

func Handler(
	reader domain.TargetsReader,
	writer domain.TargetsWriter,
	provider domain.Provider,
) bus.RequestHandler[bus.UnitType, Command] {
	return func(ctx context.Context, cmd Command) (result bus.UnitType, finalErr error) {
		target, err := reader.GetByID(ctx, domain.TargetID(cmd.ID))

		if err != nil {
			// Target not found, already deleted
			if errors.Is(err, apperr.ErrNotFound) {
				return bus.Unit, bus.Ignore(err)
			}

			return bus.Unit, err
		}

		if target.IsOutdated(cmd.Version) {
			return bus.Unit, nil
		}

		// Same as for the deployment, since the configuration can take some time, retrieve the latest
		// target version before updating its state.
		defer func() {
			target, err = reader.GetByID(ctx, domain.TargetID(cmd.ID))

			if err != nil {
				// Target not found, already deleted
				if errors.Is(err, apperr.ErrNotFound) {
					err = bus.Ignore(err)
				}

				finalErr = err
				return
			}

			target.Configured(cmd.Version, finalErr)

			if err = writer.Write(ctx, &target); err != nil {
				finalErr = err
			}

			finalErr = bus.Ignore(finalErr)
		}()

		finalErr = provider.Configure(ctx, target)

		return
	}
}
