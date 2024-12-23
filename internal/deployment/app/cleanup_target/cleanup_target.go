package cleanup_target

import (
	"context"
	"errors"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/storage"
)

// Cleanup a target and all its associated resources.
type Command struct {
	bus.AsyncCommand

	ID string `json:"target_id"`
}

func (Command) Name_() string   { return "deployment.command.cleanup_target" }
func (c Command) Group() string { return c.ID }

func Handler(
	reader domain.TargetsReader,
	writer domain.TargetsWriter,
	deploymentsReader domain.DeploymentsReader,
	provider domain.Provider,
	uow storage.UnitOfWorkFactory,
) bus.RequestHandler[bus.AsyncResult, Command] {
	return func(ctx context.Context, cmd Command) (result bus.AsyncResult, finalErr error) {
		target, err := reader.GetByID(ctx, domain.TargetID(cmd.ID))

		if err != nil {
			// If the target doesn't exist anymore, may be it has been processed by another job in rare case, so just returns
			if errors.Is(err, apperr.ErrNotFound) {
				return result, nil
			}

			return result, err
		}

		ongoing, err := deploymentsReader.HasRunningOrPendingDeploymentsOnTarget(ctx, target.ID())

		if err != nil {
			return result, err
		}

		strategy, err := target.CanBeCleaned(ongoing)

		if err != nil {
			if errors.Is(err, domain.ErrRunningOrPendingDeployments) {
				return bus.AsyncResultDelay, nil
			}

			return result, err
		}

		defer func() {
			if finalErr != nil {
				return
			}

			finalErr = uow.Create(ctx, func(ctx context.Context) error {
				if target, err = reader.GetByID(ctx, target.ID()); err != nil {
					if errors.Is(err, apperr.ErrNotFound) {
						return nil
					}

					return err
				}

				if err = target.CleanedUp(); err != nil {
					return err
				}

				return writer.Write(ctx, &target)
			})
		}()

		finalErr = provider.CleanupTarget(ctx, target, strategy)
		return
	}
}
