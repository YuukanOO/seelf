package configure_target

import (
	"context"

	"github.com/YuukanOO/seelf/internal/deployment/app"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/bus"
)

func OnTargetCreatedHandler(scheduler bus.Scheduler) bus.SignalHandler[domain.TargetCreated] {
	return func(ctx context.Context, evt domain.TargetCreated) error {
		return scheduler.Queue(ctx, Command{
			ID:      string(evt.ID),
			Version: evt.State.Version(),
		}, bus.WithGroup(app.TargetConfigurationGroup(evt.ID)), bus.WithPolicy(bus.JobPolicyMerge))
	}
}
