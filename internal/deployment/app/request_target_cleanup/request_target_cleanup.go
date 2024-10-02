package request_target_cleanup

import (
	"context"

	auth "github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/bus"
)

type Command struct {
	bus.Command[bus.UnitType]

	ID string `json:"id"`
}

func (Command) Name_() string { return "deployment.command.request_target_cleanup" }

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

		apps, err := appsReader.HasAppsOnTarget(ctx, target.ID())

		if err != nil {
			return bus.Unit, err
		}

		if err = target.RequestDelete(apps, auth.CurrentUser(ctx).MustGet()); err != nil {
			return bus.Unit, err
		}

		return bus.Unit, writer.Write(ctx, &target)
	}
}
