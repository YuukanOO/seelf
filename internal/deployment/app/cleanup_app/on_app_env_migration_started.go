package cleanup_app

import (
	"context"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/bus"
)

// When an application is migrating from a target to another, clean the target we're leaving.
func OnAppEnvMigrationStartedHandler(scheduler bus.Scheduler) bus.SignalHandler[domain.AppEnvMigrationStarted] {
	return func(ctx context.Context, evt domain.AppEnvMigrationStarted) error {
		return scheduler.Queue(ctx, Command{
			AppID:       string(evt.ID),
			TargetID:    string(evt.Migration.Target()),
			Environment: string(evt.Environment),
			From:        evt.Migration.Interval().From(),
			To:          evt.Migration.Interval().To(),
		})
	}
}
