package delete_target

import (
	"context"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/bus"
)

type Command struct {
	bus.Command[bus.UnitType]

	ID string `json:"id"`
}

func (Command) Name_() string { return "deployment.command.delete_target" }

func Handler(
	reader domain.TargetsReader,
	writer domain.TargetsWriter,
	appsReader domain.AppsReader,
) bus.RequestHandler[bus.UnitType, Command] {
	return func(ctx context.Context, cmd Command) (bus.UnitType, error) {
		target, err := reader.GetByID(ctx, domain.TargetID(cmd.ID))

		if err != nil {
			return bus.Unit, err
		}

		apps, err := appsReader.GetAppsOnTargetCount(ctx, target.ID())

		if err != nil {
			return bus.Unit, err
		}

		if err = target.Delete(apps); err != nil {
			return bus.Unit, err
		}

		if err = writer.Write(ctx, &target); err != nil {
			return bus.Unit, err
		}

		return bus.Unit, nil
	}
}
