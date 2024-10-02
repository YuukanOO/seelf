package cleanup_target

import (
	"context"
	"errors"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/bus/embedded"
)

func OnJobDismissedHandler(
	reader domain.TargetsReader,
	writer domain.TargetsWriter,
) bus.SignalHandler[embedded.JobDismissed] {
	return func(ctx context.Context, evt embedded.JobDismissed) error {
		if _, isCleanupJob := evt.Command.(Command); !isCleanupJob {
			return nil
		}

		target, err := reader.GetByID(ctx, domain.TargetID(evt.Command.ResourceID()))

		if err != nil {
			// Target deleted, no need to go further
			if errors.Is(err, apperr.ErrNotFound) {
				return nil
			}

			return err
		}

		if err = target.CleanedUp(); err != nil {
			return err
		}

		return writer.Write(ctx, &target)
	}
}
