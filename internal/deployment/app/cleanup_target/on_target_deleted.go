package cleanup_target

import (
	"context"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/bus"
)

func OnTargetDeletedHandler(provider domain.Provider) bus.SignalHandler[domain.TargetDeleted] {
	return func(ctx context.Context, evt domain.TargetDeleted) error {
		return provider.RemoveConfiguration(ctx, evt.ID)
	}
}
