package configure_target

import (
	"context"
	"errors"
	"time"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/storage"
)

type Command struct {
	bus.AsyncCommand

	ID      string    `json:"target_id"`
	Version time.Time `json:"version"`
}

func (Command) Name_() string   { return "deployment.command.configure_target" }
func (c Command) Group() string { return c.ID }

func Handler(
	reader domain.TargetsReader,
	writer domain.TargetsWriter,
	provider domain.Provider,
	uow storage.UnitOfWorkFactory,
) bus.RequestHandler[bus.AsyncResult, Command] {
	return func(ctx context.Context, cmd Command) (result bus.AsyncResult, finalErr error) {
		target, finalErr := reader.GetByID(ctx, domain.TargetID(cmd.ID))

		if finalErr != nil {
			// Target not found, already deleted
			if errors.Is(finalErr, apperr.ErrNotFound) {
				finalErr = nil
			}

			return
		}

		if target.IsOutdated(cmd.Version) {
			return
		}

		var assigned domain.TargetEntrypointsAssigned

		// Same as for the deployment, since the configuration can take some time, retrieve the latest
		// target version before updating its state in a transaction
		defer func() {
			finalErr = uow.Create(ctx, func(ctx context.Context) error {
				var err error

				if target, err = reader.GetByID(ctx, target.ID()); err != nil {
					// Target not found, already deleted
					if errors.Is(err, apperr.ErrNotFound) {
						return nil
					}

					return err
				}

				if err = target.Configured(cmd.Version, assigned, finalErr); err != nil &&
					!errors.Is(err, domain.ErrTargetConfigurationOutdated) {
					return err
				}

				return writer.Write(ctx, &target)
			})
		}()

		assigned, finalErr = provider.Setup(ctx, target)
		return
	}
}
