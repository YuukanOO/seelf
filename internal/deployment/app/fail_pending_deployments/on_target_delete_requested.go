package fail_pending_deployments

import (
	"context"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/monad"
)

func OnTargetDeleteRequestedHandler(writer domain.DeploymentsWriter) bus.SignalHandler[domain.TargetCleanupRequested] {
	return func(ctx context.Context, evt domain.TargetCleanupRequested) error {
		return writer.FailDeployments(ctx, domain.ErrTargetCleanupRequested, domain.FailCriteria{
			Status: monad.Value(domain.DeploymentStatusPending),
			Target: monad.Value(evt.ID),
		})
	}
}
