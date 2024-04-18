package fail_pending_deployments

import (
	"context"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/monad"
)

// When an app is about to be deleted, cancel all pending deployments
func OnAppCleanupRequestedHandler(writer domain.DeploymentsWriter) bus.SignalHandler[domain.AppCleanupRequested] {
	return func(ctx context.Context, evt domain.AppCleanupRequested) error {
		return writer.FailDeployments(ctx, domain.ErrAppCleanupRequested, domain.FailCriterias{
			Status: monad.Value(domain.DeploymentStatusPending),
			App:    monad.Value(evt.ID),
		})
	}
}
