package configure_target

import (
	"context"

	"github.com/YuukanOO/seelf/internal/deployment/app"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/bus"
)

func OnTargetStateChangedHandler(scheduler bus.Scheduler) bus.SignalHandler[domain.TargetStateChanged] {
	return func(ctx context.Context, evt domain.TargetStateChanged) error {
		if evt.State.Status() != domain.TargetStatusConfiguring {
			return nil
		}

		return scheduler.Queue(ctx, Command{
			ID:      string(evt.ID),
			Version: evt.State.Version(),
		}, bus.WithGroup(app.TargetConfigurationGroup(evt.ID)), bus.WithPolicy(bus.JobPolicyMerge))
	}
}
