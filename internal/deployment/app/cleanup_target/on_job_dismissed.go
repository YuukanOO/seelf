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
		cmd, isCleanupJob := evt.Command.(Command)

		if !isCleanupJob {
			return nil
		}

		target, err := reader.GetByID(ctx, domain.TargetID(cmd.ID))

		if err != nil {
			// Target deleted, no need to go further
			if errors.Is(err, apperr.ErrNotFound) {
				return nil
			}

			return err
		}

		_ = target.CleanedUp()

		return writer.Write(ctx, &target)
	}
}
