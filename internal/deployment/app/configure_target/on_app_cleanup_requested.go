package configure_target

import (
	"context"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/bus"
)

// When an application cleanup has been requested, unexpose the application from all targets.
func OnAppCleanupRequestedHandler(
	reader domain.TargetsReader,
	writer domain.TargetsWriter,
) bus.SignalHandler[domain.AppCleanupRequested] {
	return func(ctx context.Context, evt domain.AppCleanupRequested) error {
		if evt.ProductionConfig.Target() == evt.StagingConfig.Target() {
			target, err := reader.GetByID(ctx, evt.ProductionConfig.Target())

			if err != nil {
				return err
			}

			target.UnExposeEntrypoints(evt.ID)

			return writer.Write(ctx, &target)
		}

		productionTarget, err := reader.GetByID(ctx, evt.ProductionConfig.Target())

		if err != nil {
			return err
		}

		productionTarget.UnExposeEntrypoints(evt.ID, domain.Production)

		stagingTarget, err := reader.GetByID(ctx, evt.StagingConfig.Target())

		if err != nil {
			return err
		}

		stagingTarget.UnExposeEntrypoints(evt.ID, domain.Staging)

		return writer.Write(ctx, &productionTarget, &stagingTarget)
	}
}
