package configure_target

import (
	"context"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/bus"
)

func OnDeploymentStateChangedHandler(
	reader domain.TargetsReader,
	writer domain.TargetsWriter,
) bus.SignalHandler[domain.DeploymentStateChanged] {
	return func(ctx context.Context, evt domain.DeploymentStateChanged) error {
		if !evt.HasSucceeded() {
			return nil
		}

		target, err := reader.GetByID(ctx, evt.Config.Target())

		if err != nil {
			return err
		}

		target.ExposeEntrypoints(evt.ID.AppID(), evt.Config.Environment(), evt.State.Services().Get(domain.Services{}))

		return writer.Write(ctx, &target)
	}
}
