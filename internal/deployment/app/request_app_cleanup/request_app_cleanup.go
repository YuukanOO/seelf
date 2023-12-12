package request_app_cleanup

import (
	"context"

	auth "github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/bus"
)

// Mark the application for deletion.
type Command struct {
	bus.Command[bool]

	ID string `json:"-"`
}

func (Command) Name_() string { return "deployment.command.request_app_cleanup" }

func Handler(
	reader domain.AppsReader,
	writer domain.AppsWriter,
) bus.RequestHandler[bool, Command] {
	return func(ctx context.Context, cmd Command) (bool, error) {
		app, err := reader.GetByID(ctx, domain.AppID(cmd.ID))

		if err != nil {
			return false, err
		}

		app.RequestCleanup(auth.CurrentUser(ctx).MustGet())

		if err = writer.Write(ctx, &app); err != nil {
			return false, err
		}

		return true, nil
	}
}
