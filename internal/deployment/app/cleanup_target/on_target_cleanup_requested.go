package cleanup_target

import (
	"context"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/bus"
)

func OnTargetCleanupRequestedHandler(scheduler bus.Scheduler) bus.SignalHandler[domain.TargetCleanupRequested] {
	return func(ctx context.Context, evt domain.TargetCleanupRequested) error {
		return scheduler.Queue(ctx, Command{
			ID: string(evt.ID),
		})
	}
}
