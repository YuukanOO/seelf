package cleanup_app

import (
	"context"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/bus"
)

// Cleanup an application artifacts, images, networks, volumes and so on...
type Command struct {
	bus.Command[bool]

	ID string `json:"id"`
}

func (Command) Name_() string { return "deployment.command.cleanup_app" }

func Handler(
	deploymentsReader domain.DeploymentsReader,
	reader domain.AppsReader,
	writer domain.AppsWriter,
	artifactManager domain.ArtifactManager,
	backend domain.Backend,
) bus.RequestHandler[bool, Command] {
	return func(ctx context.Context, cmd Command) (bool, error) {
		app, err := reader.GetByID(ctx, domain.AppID(cmd.ID))

		if err != nil {
			return false, err
		}

		count, err := deploymentsReader.GetRunningOrPendingDeploymentsCount(ctx, app.ID())

		if err != nil {
			return false, err
		}

		// Before calling the backend cleanup, make sure the application can be safely deleted.
		if err = app.Delete(count); err != nil {
			return false, err
		}

		if err = backend.Cleanup(ctx, app); err != nil {
			return false, err
		}

		if err = artifactManager.Cleanup(ctx, app); err != nil {
			return false, err
		}

		if err = writer.Write(ctx, &app); err != nil {
			return false, err
		}

		return true, nil
	}
}
