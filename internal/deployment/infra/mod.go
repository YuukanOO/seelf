package infra

import (
	"context"
	"errors"

	"github.com/YuukanOO/seelf/internal/deployment/app/cleanup_app"
	"github.com/YuukanOO/seelf/internal/deployment/app/cleanup_target"
	"github.com/YuukanOO/seelf/internal/deployment/app/configure_target"
	"github.com/YuukanOO/seelf/internal/deployment/app/create_app"
	"github.com/YuukanOO/seelf/internal/deployment/app/create_registry"
	"github.com/YuukanOO/seelf/internal/deployment/app/create_target"
	"github.com/YuukanOO/seelf/internal/deployment/app/delete_app"
	"github.com/YuukanOO/seelf/internal/deployment/app/delete_registry"
	"github.com/YuukanOO/seelf/internal/deployment/app/delete_target"
	"github.com/YuukanOO/seelf/internal/deployment/app/deploy"
	"github.com/YuukanOO/seelf/internal/deployment/app/expose_seelf_container"
	"github.com/YuukanOO/seelf/internal/deployment/app/fail_pending_deployments"
	"github.com/YuukanOO/seelf/internal/deployment/app/get_deployment_log"
	"github.com/YuukanOO/seelf/internal/deployment/app/promote"
	"github.com/YuukanOO/seelf/internal/deployment/app/queue_deployment"
	"github.com/YuukanOO/seelf/internal/deployment/app/reconfigure_target"
	"github.com/YuukanOO/seelf/internal/deployment/app/redeploy"
	"github.com/YuukanOO/seelf/internal/deployment/app/request_app_cleanup"
	"github.com/YuukanOO/seelf/internal/deployment/app/request_target_cleanup"
	"github.com/YuukanOO/seelf/internal/deployment/app/update_app"
	"github.com/YuukanOO/seelf/internal/deployment/app/update_registry"
	"github.com/YuukanOO/seelf/internal/deployment/app/update_target"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
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
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/storage/sqlite"
)

type Options interface {
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
	registriesStore := deploymentsqlite.NewRegistriesStore(db)
	deploymentQueryHandler := deploymentsqlite.NewGateway(db)

	artifactManager := artifact.NewLocal(opts, logger)

	sourceFacade := source.NewFacade(
		raw.New(),
		archive.New(),
		git.New(appsStore),
	)

	dock := docker.New(logger)
	providerFacade := provider.NewFacade(
		dock,
	)

	bus.Register(b, expose_seelf_container.Handler(targetsStore, targetsStore, dock))
	bus.Register(b, create_app.Handler(appsStore, appsStore))
	bus.Register(b, update_app.Handler(appsStore, appsStore))
	bus.Register(b, queue_deployment.Handler(appsStore, deploymentsStore, deploymentsStore, sourceFacade))
	bus.Register(b, deploy.Handler(deploymentsStore, deploymentsStore, artifactManager, sourceFacade, providerFacade, targetsStore, registriesStore))
	bus.Register(b, request_app_cleanup.Handler(appsStore, appsStore))
	bus.Register(b, delete_app.Handler(appsStore, appsStore, artifactManager))
	bus.Register(b, cleanup_app.Handler(targetsStore, deploymentsStore, providerFacade))
	bus.Register(b, get_deployment_log.Handler(deploymentsStore, artifactManager))
	bus.Register(b, redeploy.Handler(appsStore, deploymentsStore, deploymentsStore))
	bus.Register(b, promote.Handler(appsStore, deploymentsStore, deploymentsStore))
	bus.Register(b, create_target.Handler(targetsStore, targetsStore, providerFacade))
	bus.Register(b, configure_target.Handler(targetsStore, targetsStore, providerFacade))
	bus.Register(b, reconfigure_target.Handler(targetsStore, targetsStore))
	bus.Register(b, update_target.Handler(targetsStore, targetsStore, providerFacade))
	bus.Register(b, request_target_cleanup.Handler(targetsStore, targetsStore, appsStore))
	bus.Register(b, cleanup_target.Handler(targetsStore, deploymentsStore, providerFacade))
	bus.Register(b, delete_target.Handler(targetsStore, targetsStore, providerFacade))
	bus.Register(b, create_registry.Handler(registriesStore, registriesStore))
	bus.Register(b, update_registry.Handler(registriesStore, registriesStore))
	bus.Register(b, delete_registry.Handler(registriesStore, registriesStore))
	bus.Register(b, deploymentQueryHandler.GetAllApps)
	bus.Register(b, deploymentQueryHandler.GetAppByID)
	bus.Register(b, deploymentQueryHandler.GetAllDeploymentsByApp)
	bus.Register(b, deploymentQueryHandler.GetDeploymentByID)
	bus.Register(b, deploymentQueryHandler.GetAllTargets)
	bus.Register(b, deploymentQueryHandler.GetTargetByID)
	bus.Register(b, deploymentQueryHandler.GetRegistries)
	bus.Register(b, deploymentQueryHandler.GetRegistryByID)

	bus.On(b, deploy.OnDeploymentCreatedHandler(scheduler))
	bus.On(b, redeploy.OnAppEnvChangedHandler(appsStore, deploymentsStore, deploymentsStore))
	bus.On(b, delete_app.OnAppCleanupRequestedHandler(scheduler))
	bus.On(b, cleanup_app.OnAppEnvChangedHandler(scheduler))
	bus.On(b, cleanup_app.OnAppCleanupRequestedHandler(scheduler))
	bus.On(b, fail_pending_deployments.OnTargetDeleteRequestedHandler(deploymentsStore))
	bus.On(b, fail_pending_deployments.OnAppCleanupRequestedHandler(deploymentsStore))
	bus.On(b, fail_pending_deployments.OnAppEnvChangedHandler(deploymentsStore))
	bus.On(b, cleanup_target.OnTargetCleanupRequestedHandler(scheduler))
	bus.On(b, configure_target.OnTargetCreatedHandler(scheduler))
	bus.On(b, configure_target.OnTargetStateChangedHandler(scheduler))
	bus.On(b, configure_target.OnDeploymentStateChangedHandler(targetsStore, targetsStore))
	bus.On(b, configure_target.OnAppEnvChangedHandler(targetsStore, targetsStore))
	bus.On(b, configure_target.OnAppCleanupRequestedHandler(targetsStore, targetsStore))
	bus.On(b, delete_target.OnTargetCleanupRequestedHandler(scheduler))

	if err := db.Migrate(deploymentsqlite.Migrations); err != nil {
		return err
	}

	// Fail running deployments in case of a hard reset.
	return deploymentsStore.FailDeployments(context.Background(), errors.New("server_reset"), domain.FailCriteria{
		Status: monad.Value(domain.DeploymentStatusRunning),
	})
}
