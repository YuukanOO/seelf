package cleanup_app

import (
	"context"
	"errors"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/bus"
)

// Cleanup an application artifacts, images, networks, volumes and so on...
type Command struct {
	bus.Command[bus.UnitType]

	ID string `json:"id"`
}

func (Command) Name_() string { return "deployment.command.cleanup_app" }

func Handler(
	deploymentsReader domain.DeploymentsReader,
	reader domain.AppsReader,
	writer domain.AppsWriter,
	artifactManager domain.ArtifactManager,
	backend domain.Backend,
) bus.RequestHandler[bus.UnitType, Command] {
	return func(ctx context.Context, cmd Command) (bus.UnitType, error) {
		app, err := reader.GetByID(ctx, domain.AppID(cmd.ID))

		if err != nil {
			// If the application doesn't exist anymore, may be it has been processed by another job in rare case, so just returns
			if errors.Is(err, apperr.ErrNotFound) {
				return bus.Unit, nil
			}

			return bus.Unit, err
		}

		count, err := deploymentsReader.GetRunningOrPendingDeploymentsCount(ctx, app.ID())

		if err != nil {
			return bus.Unit, err
		}

		// Before calling the backend cleanup, make sure the application can be safely deleted.
		if err = app.Delete(count); err != nil {
			return bus.Unit, err
		}

		if err = backend.Cleanup(ctx, app); err != nil {
			return bus.Unit, err
		}

		if err = artifactManager.Cleanup(ctx, app); err != nil {
			return bus.Unit, err
		}

		if err = writer.Write(ctx, &app); err != nil {
			return bus.Unit, err
		}

		return bus.Unit, nil
	}
}
