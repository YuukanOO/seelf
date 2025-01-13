package cleanup_app

import (
	"context"
	"errors"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/bus/embedded"
)

func OnJobDismissedHandler(
	reader domain.AppsReader,
	writer domain.AppsWriter,
) bus.SignalHandler[embedded.JobDismissed] {
	return func(ctx context.Context, evt embedded.JobDismissed) error {
		cmd, isCleanupJob := evt.Command.(Command)

		if !isCleanupJob {
			return nil
		}

		app, err := reader.GetByID(ctx, domain.AppID(cmd.AppID))

		if err != nil {
			// App deleted, no need to go further
			if errors.Is(err, apperr.ErrNotFound) {
				return nil
			}

			return err
		}

		_ = app.CleanedUp(domain.EnvironmentName(cmd.Environment), domain.TargetID(cmd.TargetID))

		return writer.Write(ctx, &app)
	}
}
