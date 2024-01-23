package delete_app

import (
	"context"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/bus"
)

// Upon receiving a cleanup request, queue a job to remove everything related to the application.
func OnAppCleanupRequestedHandler(scheduler bus.Scheduler) bus.SignalHandler[domain.AppCleanupRequested] {
	return func(ctx context.Context, evt domain.AppCleanupRequested) error {
		return scheduler.Queue(ctx, Command{
			ID: string(evt.ID),
		}, bus.WithPolicy(bus.JobPolicyWaitForOthersResourceID))
	}
}
