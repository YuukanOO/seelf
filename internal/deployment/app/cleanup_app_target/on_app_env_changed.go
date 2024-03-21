package cleanup_app_target

import (
	"context"

	"github.com/YuukanOO/seelf/internal/deployment/app"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/bus"
)

func OnAppEnvChanged(scheduler bus.Scheduler) bus.SignalHandler[domain.AppEnvChanged] {
	return func(ctx context.Context, evt domain.AppEnvChanged) error {
		// No target change, nothing to do
		if evt.Config.Target() == evt.OldTarget {
			return nil
		}

		return scheduler.Queue(ctx, Command{
			AppID:       string(evt.ID),
			TargetID:    string(evt.OldTarget),
			Environment: string(evt.Environment),
		}, bus.WithDedupeName(app.AppCleanupDedupeName(evt.ID)))
	}
}
