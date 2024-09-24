package configure_target

import (
	"context"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/bus"
)

// When an application cleanup has been requested, un-expose the application from all targets.
func OnAppCleanupRequestedHandler(
	reader domain.TargetsReader,
	writer domain.TargetsWriter,
) bus.SignalHandler[domain.AppCleanupRequested] {
	return func(ctx context.Context, evt domain.AppCleanupRequested) error {
		if err := unExpose(ctx, reader, writer, evt.ProductionConfig.Target(), evt.ID); err != nil {
			return err
		}

		if evt.ProductionConfig.Target() == evt.StagingConfig.Target() {
			return nil
		}

		return unExpose(ctx, reader, writer, evt.StagingConfig.Target(), evt.ID)
	}
}

func unExpose(
	ctx context.Context,
	reader domain.TargetsReader,
	writer domain.TargetsWriter,
	id domain.TargetID,
	app domain.AppID,
) error {
	target, err := reader.GetByID(ctx, id)

	if err != nil {
		return err
	}

	target.UnExposeEntrypoints(app)

	return writer.Write(ctx, &target)
}
