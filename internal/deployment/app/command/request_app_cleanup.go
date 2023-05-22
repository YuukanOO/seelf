package command

import (
	"context"

	auth "github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
)

type RequestAppCleanupCommand struct {
	ID string `json:"-"`
}

// Mark the application for deletion.
func RequestAppCleanup(
	reader domain.AppsReader,
	writer domain.AppsWriter,
) func(context.Context, RequestAppCleanupCommand) error {
	return func(ctx context.Context, cmd RequestAppCleanupCommand) error {
		app, err := reader.GetByID(ctx, domain.AppID(cmd.ID))

		if err != nil {
			return err
		}

		app.RequestCleanup(auth.CurrentUser(ctx).MustGet())

		return writer.Write(ctx, &app)
	}
}
