package configure_target

import (
	"context"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/bus"
)

// When an application environment has changed and only if the target has changed,
// unexpose the application from the old target.
func OnAppEnvChangedHandler(
	reader domain.TargetsReader,
	writer domain.TargetsWriter,
) bus.SignalHandler[domain.AppEnvChanged] {
	return func(ctx context.Context, evt domain.AppEnvChanged) error {
		if !evt.TargetHasChanged() {
			return nil
		}

		target, err := reader.GetByID(ctx, evt.OldConfig.Target())

		if err != nil {
			return err
		}

		target.UnExposeEntrypoints(evt.ID, evt.Environment)

		return writer.Write(ctx, &target)
	}
}
