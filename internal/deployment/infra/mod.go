package infra

import (
	"context"
	"errors"

	"github.com/YuukanOO/seelf/internal/deployment/app/cleanup_app"
	"github.com/YuukanOO/seelf/internal/deployment/app/cleanup_app_target"
	"github.com/YuukanOO/seelf/internal/deployment/app/cleanup_target"
	"github.com/YuukanOO/seelf/internal/deployment/app/configure_target"
	"github.com/YuukanOO/seelf/internal/deployment/app/create_app"
	"github.com/YuukanOO/seelf/internal/deployment/app/create_target"
	"github.com/YuukanOO/seelf/internal/deployment/app/deploy"
	"github.com/YuukanOO/seelf/internal/deployment/app/fail_pending_deployments"
	"github.com/YuukanOO/seelf/internal/deployment/app/fail_running_deployments"
	"github.com/YuukanOO/seelf/internal/deployment/app/get_deployment_log"
	"github.com/YuukanOO/seelf/internal/deployment/app/promote"
	"github.com/YuukanOO/seelf/internal/deployment/app/queue_deployment"
	"github.com/YuukanOO/seelf/internal/deployment/app/redeploy"
	"github.com/YuukanOO/seelf/internal/deployment/app/request_app_cleanup"
	"github.com/YuukanOO/seelf/internal/deployment/app/request_target_delete"
	"github.com/YuukanOO/seelf/internal/deployment/app/update_app"
	"github.com/YuukanOO/seelf/internal/deployment/app/update_target"
	"github.com/YuukanOO/seelf/internal/deployment/infra/artifact"
	"github.com/YuukanOO/seelf/internal/deployment/infra/provider"
	"github.com/YuukanOO/seelf/internal/deployment/infra/provider/docker"
	"github.com/YuukanOO/seelf/internal/deployment/infra/source"
	"github.com/YuukanOO/seelf/internal/deployment/infra/source/archive"
	"github.com/YuukanOO/seelf/internal/deployment/infra/source/git"
	"github.com/YuukanOO/seelf/internal/deployment/infra/source/raw"
	deploymentsqlite "github.com/YuukanOO/seelf/internal/deployment/infra/sqlite"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/log"
	"github.com/YuukanOO/seelf/pkg/storage/sqlite"
)

type Options interface {
	docker.Options
	artifact.LocalOptions
}

// Setup the deployment module and register everything needed in the given
// bus.
func Setup(
	opts Options,
	logger log.Logger,
	db *sqlite.Database,
	b bus.Bus,
	scheduler bus.Scheduler,
) error {
	appsStore := deploymentsqlite.NewAppsStore(db)
	deploymentsStore := deploymentsqlite.NewDeploymentsStore(db)
	targetsStore := deploymentsqlite.NewTargetsStore(db)
	deploymentQueryHandler := deploymentsqlite.NewGateway(db)

	artifactManager := artifact.NewLocal(opts, logger)

	sourceFacade := source.NewFacade(
		raw.New(),
		archive.New(),
		git.New(appsStore),
	)

	providerFacade := provider.NewFacade(
		docker.New(opts, logger),
	)

	bus.Register(b, create_app.Handler(appsStore, appsStore))
	bus.Register(b, update_app.Handler(appsStore, appsStore))
	bus.Register(b, queue_deployment.Handler(appsStore, deploymentsStore, deploymentsStore, sourceFacade))
	bus.Register(b, deploy.Handler(deploymentsStore, deploymentsStore, artifactManager, sourceFacade, providerFacade, targetsStore))
	bus.Register(b, fail_running_deployments.Handler(deploymentsStore))
	bus.Register(b, request_app_cleanup.Handler(appsStore, appsStore))
	bus.Register(b, cleanup_app.Handler(deploymentsStore, appsStore, appsStore, artifactManager, providerFacade, targetsStore))
	bus.Register(b, cleanup_app_target.Handler(targetsStore, providerFacade))
	bus.Register(b, get_deployment_log.Handler(deploymentsStore, artifactManager))
	bus.Register(b, redeploy.Handler(appsStore, deploymentsStore, deploymentsStore))
	bus.Register(b, promote.Handler(appsStore, deploymentsStore, deploymentsStore))
	bus.Register(b, create_target.Handler(targetsStore, targetsStore, providerFacade))
	bus.Register(b, configure_target.Handler(targetsStore, targetsStore, providerFacade))
	bus.Register(b, update_target.Handler(targetsStore, targetsStore, providerFacade))
	bus.Register(b, request_target_delete.Handler(targetsStore, targetsStore, appsStore))
	bus.Register(b, cleanup_target.Handler(targetsStore, targetsStore, deploymentsStore, providerFacade))
	bus.Register(b, deploymentQueryHandler.GetAllApps)
	bus.Register(b, deploymentQueryHandler.GetAppByID)
	bus.Register(b, deploymentQueryHandler.GetAllDeploymentsByApp)
	bus.Register(b, deploymentQueryHandler.GetDeploymentByID)
	bus.Register(b, deploymentQueryHandler.GetAllTargets)
	bus.Register(b, deploymentQueryHandler.GetTargetByID)

	bus.On(b, deploy.OnDeploymentCreatedHandler(scheduler))
	bus.On(b, redeploy.OnAppEnvChanged(appsStore, deploymentsStore, deploymentsStore))
	bus.On(b, cleanup_app.OnAppCleanupRequestedHandler(scheduler))
	bus.On(b, cleanup_app_target.OnAppEnvChanged(scheduler))
	bus.On(b, fail_pending_deployments.OnAppCleanupRequestedHandler(deploymentsStore))
	bus.On(b, cleanup_target.OnTargetDeleteRequested(scheduler))
	bus.On(b, configure_target.OnTargetCreatedHandler(scheduler))
	bus.On(b, configure_target.OnTargetStateChanged(scheduler))

	if err := db.Migrate(deploymentsqlite.Migrations); err != nil {
		return err
	}

	// Fail running jobs in case seelf has been hard stopped
	_, err := bus.Send(b, context.Background(), fail_running_deployments.Command{
		Reason: errors.New("server_reset"),
	})

	return err
}
