package configure_target

import (
	"context"
	"errors"
	"time"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/bus"
)

type Command struct {
	bus.AsyncCommand

	ID      string    `json:"id"`
	Version time.Time `json:"version"`
}

func (Command) Name_() string        { return "deployment.command.configure_target" }
func (c Command) ResourceID() string { return c.ID }
func (c Command) Group() string      { return c.ID }

func Handler(
	reader domain.TargetsReader,
	writer domain.TargetsWriter,
	provider domain.Provider,
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
		// target version before updating its state.
		defer func() {
			var err error

			if target, err = reader.GetByID(ctx, target.ID()); err != nil {
				// Target not found, already deleted
				if errors.Is(err, apperr.ErrNotFound) {
					err = nil
				}

				finalErr = err
				return
			}

			if err = target.Configured(cmd.Version, assigned, finalErr); err != nil &&
				!errors.Is(err, domain.ErrTargetConfigurationOutdated) {
				finalErr = err
				return
			}

			finalErr = writer.Write(ctx, &target)
		}()

		assigned, finalErr = provider.Setup(ctx, target)
		return
	}
}
