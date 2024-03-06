package cleanup_target

import (
	"context"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/monad"
)

func OnTargetDeleteRequested(scheduler bus.Scheduler) bus.SignalHandler[domain.TargetDeleteRequested] {
	return func(ctx context.Context, evt domain.TargetDeleteRequested) error {
		return scheduler.Queue(ctx, Command{
			ID: string(evt.ID),
		}, monad.None[string]())
	}
}
