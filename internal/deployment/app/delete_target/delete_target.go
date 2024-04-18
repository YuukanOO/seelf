package delete_target

import (
	"context"
	"errors"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/bus"
)

type Command struct {
	bus.Command[bus.UnitType]

	ID string `json:"id"`
}

func (Command) Name_() string        { return "deployment.command.delete_target" }
func (c Command) ResourceID() string { return c.ID }

func Handler(
	reader domain.TargetsReader,
	writer domain.TargetsWriter,
	provider domain.Provider,
) bus.RequestHandler[bus.UnitType, Command] {
	return func(ctx context.Context, cmd Command) (bus.UnitType, error) {
		target, err := reader.GetByID(ctx, domain.TargetID(cmd.ID))

		if err != nil {
			if errors.Is(err, apperr.ErrNotFound) {
				return bus.Unit, nil
			}

			return bus.Unit, err
		}

		// Resources have been cleaned up here thanks to the scheduler policy
		if err = target.Delete(true); err != nil {
			return bus.Unit, err
		}

		// Either way, remove eventual configuration tied to the target
		if err = provider.RemoveConfiguration(ctx, target); err != nil {
			return bus.Unit, err
		}

		return bus.Unit, writer.Write(ctx, &target)
	}
}
