package command

import (
	"context"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
)

type CleanupAppCommand struct {
	ID string
}

// Cleanup an application artifacts, images, networks, volumes and so on...
func CleanupApp(
	deploymentsReader domain.DeploymentsReader,
	reader domain.AppsReader,
	writer domain.AppsWriter,
	artifactManager domain.ArtifactManager,
	backend domain.Backend,
) func(context.Context, CleanupAppCommand) error {
	return func(ctx context.Context, cmd CleanupAppCommand) error {
		app, err := reader.GetByID(ctx, domain.AppID(cmd.ID))

		if err != nil {
			return err
		}

		count, err := deploymentsReader.GetRunningOrPendingDeploymentsCount(ctx, app.ID())

		if err != nil {
			return err
		}

		// Before calling the backend cleanup, make sure the application can be safely deleted.
		if err = app.Delete(count); err != nil {
			return err
		}

		if err = backend.Cleanup(ctx, app); err != nil {
			return err
		}

		if err = artifactManager.Cleanup(ctx, app); err != nil {
			return err
		}

		return writer.Write(ctx, &app)
	}
}
