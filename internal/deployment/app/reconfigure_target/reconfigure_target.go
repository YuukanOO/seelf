package reconfigure_target

import (
	"context"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/bus"
)

type Command struct {
	bus.Command[bus.UnitType]

	ID string `json:"id"`
}

func (Command) Name_() string { return "deployment.command.reconfigure_target" }

func Handler(
	reader domain.TargetsReader,
	writer domain.TargetsWriter,
) bus.RequestHandler[bus.UnitType, Command] {
	return func(ctx context.Context, cmd Command) (bus.UnitType, error) {
		target, err := reader.GetByID(ctx, domain.TargetID(cmd.ID))

		if err != nil {
			return bus.Unit, err
		}

		if err = target.Reconfigure(); err != nil {
			return bus.Unit, err
		}

		return bus.Unit, writer.Write(ctx, &target)
	}
}
