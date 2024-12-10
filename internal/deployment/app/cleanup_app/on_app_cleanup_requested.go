package cleanup_app

import (
	"context"
	"time"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/bus"
)

// Queue the provider cleanup stuff for both production and staging for the app being deleted.
func OnAppCleanupRequestedHandler(scheduler bus.Scheduler) bus.SignalHandler[domain.AppCleanupRequested] {
	return func(ctx context.Context, evt domain.AppCleanupRequested) error {
		now := time.Now().UTC()

		return scheduler.Queue(ctx,
			Command{
				AppID:       string(evt.ID),
				Environment: string(domain.Production),
				TargetID:    string(evt.ProductionConfig.Target()),
				From:        evt.ProductionConfig.Version(),
				To:          now,
			},
			Command{
				AppID:       string(evt.ID),
				Environment: string(domain.Staging),
				TargetID:    string(evt.StagingConfig.Target()),
				From:        evt.StagingConfig.Version(),
				To:          now,
			},
		)
	}
}
