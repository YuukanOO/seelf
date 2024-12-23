package fail_pending_deployments

import (
	"context"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/monad"
)

// When an application environment migration has started, just fail the old deployment right now.
func OnAppEnvMigrationStartedHandler(writer domain.DeploymentsWriter) bus.SignalHandler[domain.AppEnvMigrationStarted] {
	return func(ctx context.Context, evt domain.AppEnvMigrationStarted) error {
		return writer.FailDeployments(ctx, domain.ErrAppTargetChanged, domain.FailCriteria{
			Status:      monad.Value(domain.DeploymentStatusPending),
			App:         monad.Value(evt.ID),
			Environment: monad.Value(evt.Environment),
			Target:      monad.Value(evt.Migration.Target()),
		})
	}
}
