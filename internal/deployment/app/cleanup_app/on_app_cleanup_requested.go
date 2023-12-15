package cleanup_app

import (
	"context"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/monad"
)

// Upon receiving a cleanup request, queue a job to remove everything related to the application.
func OnAppCleanupRequestedHandler(scheduler bus.Scheduler) bus.SignalHandler[domain.AppCleanupRequested] {
	return func(ctx context.Context, evt domain.AppCleanupRequested) error {
		cmd := Command{
			ID: string(evt.ID),
		}

		return scheduler.Queue(ctx, cmd, monad.None[string](), bus.JobErrPolicyRetry)
	}
}

func init() {
	bus.RegisterForMarshalling[Command]()
}
