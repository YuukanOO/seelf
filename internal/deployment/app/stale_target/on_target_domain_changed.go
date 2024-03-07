package stale_target

import (
	"context"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/bus"
)

// On target domain changed, mark the target has stale so it will be reinitialized upon the next deployment
func OnTargetDomainChanged(provider domain.Provider) bus.SignalHandler[domain.TargetDomainChanged] {
	return func(ctx context.Context, evt domain.TargetDomainChanged) error {
		return provider.Stale(ctx, evt.ID)
	}
}
