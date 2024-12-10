package request_app_cleanup

import (
	"context"

	auth "github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/bus"
)

// Mark the application for deletion.
type Command struct {
	bus.Command[bus.UnitType]

	ID string `json:"-"`
}

func (Command) Name_() string { return "deployment.command.request_app_cleanup" }

func Handler(
	reader domain.AppsReader,
	writer domain.AppsWriter,
) bus.RequestHandler[bus.UnitType, Command] {
	return func(ctx context.Context, cmd Command) (bus.UnitType, error) {
		app, err := reader.GetByID(ctx, domain.AppID(cmd.ID))

		if err != nil {
			return bus.Unit, err
		}

		app.RequestDelete(auth.CurrentUser(ctx).MustGet())

		return bus.Unit, writer.Write(ctx, &app)
	}
}
