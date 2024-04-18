package cleanup_app

import (
	"context"
	"time"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/bus"
)

func OnAppEnvChangedHandler(scheduler bus.Scheduler) bus.SignalHandler[domain.AppEnvChanged] {
	return func(ctx context.Context, evt domain.AppEnvChanged) error {
		// No target change, nothing to do
		if evt.Config.Target() == evt.OldConfig.Target() {
			return nil
		}

		return scheduler.Queue(ctx, Command{
			AppID:       string(evt.ID),
			TargetID:    string(evt.OldConfig.Target()),
			Environment: string(evt.Environment),
			From:        evt.OldConfig.Version(),
			To:          time.Now().UTC(),
		}, bus.WithPolicy(bus.JobPolicyCancellable))
	}
}
