package delete_app

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

func (Command) Name_() string        { return "deployment.command.delete_app" }
func (c Command) ResourceID() string { return c.ID }

func Handler(
	reader domain.AppsReader,
	writer domain.AppsWriter,
	artifactManager domain.ArtifactManager,
) bus.RequestHandler[bus.UnitType, Command] {
	return func(ctx context.Context, cmd Command) (bus.UnitType, error) {
		app, err := reader.GetByID(ctx, domain.AppID(cmd.ID))

		if err != nil {
			if errors.Is(err, apperr.ErrNotFound) {
				return bus.Unit, nil
			}

			return bus.Unit, err
		}

		// Resources have been cleaned up here thanks to the scheduler policy
		if err = app.Delete(true); err != nil {
			return bus.Unit, err
		}

		if err = artifactManager.Cleanup(ctx, app.ID()); err != nil {
			return bus.Unit, err
		}

		return bus.Unit, writer.Write(ctx, &app)
	}
}
