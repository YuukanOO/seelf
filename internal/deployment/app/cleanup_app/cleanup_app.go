package cleanup_app

import (
	"context"
	"database/sql/driver"
	"errors"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/storage"
)

// Cleanup an application artifacts, images, networks, volumes and so on...
type Command struct {
	bus.Command[bus.UnitType]

	ID string `json:"id"`
}

func (Command) Name_() string                  { return "deployment.command.cleanup_app" }
func (c Command) Value() (driver.Value, error) { return storage.ValueJSON(c) }

func init() {
	bus.Marshallable.Register(Command{}, func(s string) (bus.Request, error) { return storage.UnmarshalJSON[Command](s) })
}

func Handler(
	deploymentsReader domain.DeploymentsReader,
	reader domain.AppsReader,
	writer domain.AppsWriter,
	artifactManager domain.ArtifactManager,
	provider domain.Provider,
	targetsReader domain.TargetsReader,
) bus.RequestHandler[bus.UnitType, Command] {
	return func(ctx context.Context, cmd Command) (bus.UnitType, error) {
		app, err := reader.GetByID(ctx, domain.AppID(cmd.ID))

		if err != nil {
			// If the application doesn't exist anymore, may be it has been processed by another job in rare case, so just returns
			if errors.Is(err, apperr.ErrNotFound) {
				return bus.Unit, bus.Ignore(err)
			}

			return bus.Unit, err
		}

		count, err := deploymentsReader.GetRunningOrPendingDeploymentsCount(
			ctx,
			app.ID(),
			domain.GetDeploymentsCountFilters{},
		)

		if err != nil {
			return bus.Unit, err
		}

		// Before calling the provider cleanup, make sure the application can be safely deleted.
		if err = app.Delete(count); err != nil {
			return bus.Unit, err
		}

		if err := handleCleanup(
			ctx,
			targetsReader,
			provider,
			app.ID(),
			app.Production().Target(),
			domain.Production,
			count,
		); err != nil {
			return bus.Unit, err
		}

		if err := handleCleanup(
			ctx,
			targetsReader,
			provider,
			app.ID(),
			app.Staging().Target(),
			domain.Staging,
			count,
		); err != nil {
			return bus.Unit, err
		}

		if err = artifactManager.Cleanup(ctx, app); err != nil {
			return bus.Unit, err
		}

		return bus.Unit, writer.Write(ctx, &app)
	}
}

func handleCleanup(
	ctx context.Context,
	reader domain.TargetsReader,
	provider domain.Provider,
	app domain.AppID,
	id domain.TargetID,
	env domain.Environment,
	count domain.RunningOrPendingAppDeploymentsCount,
) error {
	target, err := reader.GetByID(ctx, id)

	if err != nil {
		// The target does not exist anymore, it should have been deleted and cleanup up,
		// nothing to do anymore.
		if errors.Is(err, apperr.ErrNotFound) {
			return nil
		}

		return err
	}

	strategy, err := target.AppCleanupAllowed(count)

	if err != nil {
		return err
	}

	if strategy == domain.TargetCleanupStrategySkip {
		return nil
	}

	return provider.Cleanup(ctx, app, target, env)
}
