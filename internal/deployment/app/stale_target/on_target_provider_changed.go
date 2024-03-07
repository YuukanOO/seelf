package stale_target

import (
	"context"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/bus"
)

// On target provider changed, mark the target has stale so it will be reinitialized upon the next deployment
func OnTargetProviderChanged(provider domain.Provider) bus.SignalHandler[domain.TargetProviderChanged] {
	return func(ctx context.Context, evt domain.TargetProviderChanged) error {
		return provider.Stale(ctx, evt.ID)
	}
}
