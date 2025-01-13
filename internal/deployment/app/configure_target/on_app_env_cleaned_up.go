package configure_target

import (
	"context"
	"errors"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/bus"
)

// When an application has been cleaned up on a target, remove target entrypoints related to it.
func OnAppEnvCleanedUpHandler(
	reader domain.TargetsReader,
	writer domain.TargetsWriter,
) bus.SignalHandler[domain.AppEnvCleanedUp] {
	return func(ctx context.Context, evt domain.AppEnvCleanedUp) error {
		target, err := reader.GetByID(ctx, evt.Target)

		if err != nil {
			if errors.Is(err, apperr.ErrNotFound) {
				return nil
			}

			return err
		}

		target.UnExposeEntrypoints(evt.ID, evt.Environment)

		return writer.Write(ctx, &target)
	}
}
