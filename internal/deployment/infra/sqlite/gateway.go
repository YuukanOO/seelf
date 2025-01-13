package sqlite

import (
	"context"

	"github.com/YuukanOO/seelf/internal/deployment/app"
	"github.com/YuukanOO/seelf/internal/deployment/app/get_app_deployments"
	"github.com/YuukanOO/seelf/internal/deployment/app/get_app_detail"
	"github.com/YuukanOO/seelf/internal/deployment/app/get_apps"
	"github.com/YuukanOO/seelf/internal/deployment/app/get_deployment"
	"github.com/YuukanOO/seelf/internal/deployment/app/get_registries"
	"github.com/YuukanOO/seelf/internal/deployment/app/get_registry"
	"github.com/YuukanOO/seelf/internal/deployment/app/get_target"
	"github.com/YuukanOO/seelf/internal/deployment/app/get_targets"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/storage"
	"github.com/YuukanOO/seelf/pkg/storage/sqlite"
	"github.com/YuukanOO/seelf/pkg/storage/sqlite/builder"
)

type Gateway struct {
	db *sqlite.Database
}

func NewGateway(db *sqlite.Database) *Gateway {
	return &Gateway{db}
}

func (s *Gateway) GetAllApps(ctx context.Context, cmd get_apps.Query) ([]get_apps.App, error) {
	return builder.
		Query[get_apps.App](`
			SELECT
				app.id
				,app.name
				,app.cleanup_requested_at
				,cleaner.id
				,cleaner.email
				,app.created_at
				,creator.id
				,creator.email
				,production_target.id
				,production_target.name
				,production_target.url
				,staging_target.id
				,staging_target.name
				,staging_target.url
			FROM [deployment.apps] app
			INNER JOIN [auth.users] creator ON creator.id = app.created_by
			INNER JOIN [deployment.targets] AS production_target ON production_target.id = app.production_config_target
			INNER JOIN [deployment.targets] AS staging_target ON staging_target.id = app.staging_config_target
			LEFT JOIN [auth.users] cleaner ON cleaner.id = app.cleanup_requested_by`).
		All(s.db, ctx, appDataMapper, getDeploymentDataloader)
}

func (s *Gateway) GetAppByID(ctx context.Context, cmd get_app_detail.Query) (get_app_detail.App, error) {
	return builder.
		Query[get_app_detail.App](`
			SELECT
				app.id
				,app.name
				,app.version_control_url
				,app.version_control_token
				,production_target.id
				,production_target.name
				,production_target.url
				,production_migration_target.id
				,production_migration_target.name
				,production_migration_target.url
				,app.production_config_vars
				,staging_target.id
				,staging_target.name
				,staging_target.url
				,staging_migration_target.id
				,staging_migration_target.name
				,staging_migration_target.url
				,app.staging_config_vars
				,app.cleanup_requested_at
				,cleaner.id
				,cleaner.email
				,app.created_at
				,creator.id
				,creator.email
			FROM [deployment.apps] app
			INNER JOIN [auth.users] creator ON creator.id = app.created_by
			INNER JOIN [deployment.targets] production_target ON production_target.id = app.production_config_target
			LEFT JOIN [deployment.targets] production_migration_target ON production_migration_target.id = app.production_migration_target
			INNER JOIN [deployment.targets] staging_target ON staging_target.id = app.staging_config_target
			LEFT JOIN [deployment.targets] staging_migration_target ON staging_migration_target.id = app.staging_migration_target
			LEFT JOIN [auth.users] cleaner ON cleaner.id = app.cleanup_requested_by
			WHERE app.id = ?`, cmd.ID).
		One(s.db, ctx, appDetailDataMapper, getDeploymentDetailDataloader)
}

func (s *Gateway) GetAllDeploymentsByApp(ctx context.Context, cmd get_app_deployments.Query) (storage.Paginated[get_app_deployments.Deployment], error) {
	return builder.
		Select[get_app_deployments.Deployment](`
			deployment.app_id
			,deployment.deployment_number
			,deployment.config_environment
			,deployment.config_target
			,target.name
			,target.url
			,target.state_status
			,deployment.source_discriminator
			,deployment.source
			,deployment.state_status
			,deployment.state_errcode
			,deployment.state_started_at
			,deployment.state_finished_at
			,deployment.requested_at
			,requester.id
			,requester.email
			,'' -- only to use the same mapper as the latest deployments`).
		F(`
			FROM [deployment.deployments] deployment
			INNER JOIN [auth.users] requester ON requester.id = deployment.requested_by
			LEFT JOIN [deployment.targets] target ON target.id = deployment.config_target
			WHERE deployment.app_id = ?`, cmd.AppID).
		S(builder.MaybeValue(cmd.Environment, "AND deployment.config_environment = ?")).
		F("ORDER BY deployment.deployment_number DESC").
		Paginate(s.db, ctx, deploymentMapper(nil), cmd.Page.Get(1), 5)
}

func (s *Gateway) GetDeploymentByID(ctx context.Context, cmd get_deployment.Query) (get_deployment.Deployment, error) {
	return builder.
		Query[get_deployment.Deployment](`
		SELECT
			deployment.app_id
			,deployment.deployment_number
			,deployment.config_environment
			,deployment.config_target
			,target.name
			,target.url
			,target.state_status
			,target.entrypoints
			,deployment.source_discriminator
			,deployment.source
			,deployment.state_status
			,deployment.state_errcode
			,deployment.state_services
			,deployment.state_started_at
			,deployment.state_finished_at
			,deployment.requested_at
			,requester.id
			,requester.email
			,'' -- only to use the same mapper as the latest deployments
		FROM [deployment.deployments] deployment
		INNER JOIN [auth.users] requester ON requester.id = deployment.requested_by
		LEFT JOIN [deployment.targets] target ON target.id = deployment.config_target
		WHERE deployment.app_id = ? AND deployment.deployment_number = ?`, cmd.AppID, cmd.DeploymentNumber).
		One(s.db, ctx, deploymentDetailMapper(nil))
}

func (s *Gateway) GetAllTargets(ctx context.Context, cmd get_targets.Query) ([]get_target.Target, error) {
	return builder.
		Query[get_target.Target](`
		SELECT
			target.id
			,target.name
			,target.url
			,target.provider_kind
			,target.provider
			,target.state_status
			,target.state_errcode
			,target.state_last_ready_version
			,target.cleanup_requested_at
			,cleaner.id
			,cleaner.email
			,target.created_at
			,creator.id
			,creator.email
		FROM [deployment.targets] target
		INNER JOIN [auth.users] creator ON creator.id = target.created_by
		LEFT JOIN [auth.users] cleaner ON cleaner.id = target.cleanup_requested_by
		WHERE TRUE
		`).
		S(builder.If(cmd.ActiveOnly, "AND target.cleanup_requested_at IS NULL")).
		All(s.db, ctx, targetMapper)
}

func (s *Gateway) GetTargetByID(ctx context.Context, cmd get_target.Query) (get_target.Target, error) {
	return builder.
		Query[get_target.Target](`
		SELECT
			target.id
			,target.name
			,target.url
			,target.provider_kind
			,target.provider
			,target.state_status
			,target.state_errcode
			,target.state_last_ready_version
			,target.cleanup_requested_at
			,cleaner.id
			,cleaner.email
			,target.created_at
			,creator.id
			,creator.email
		FROM [deployment.targets] target
		INNER JOIN [auth.users] creator ON creator.id = target.created_by
		LEFT JOIN [auth.users] cleaner ON cleaner.id = target.cleanup_requested_by
		WHERE target.id = ?`, cmd.ID).
		One(s.db, ctx, targetMapper)
}

func (s *Gateway) GetRegistries(ctx context.Context, cmd get_registries.Query) ([]get_registry.Registry, error) {
	return builder.
		Query[get_registry.Registry](`
		SELECT
			registry.id
			,registry.name
			,registry.url
			,registry.credentials_username
			,registry.credentials_password
			,registry.created_at
			,creator.id
			,creator.email
		FROM [deployment.registries] registry
		INNER JOIN [auth.users] creator ON creator.id = registry.created_by`).
		All(s.db, ctx, registryMapper)
}

func (s *Gateway) GetRegistryByID(ctx context.Context, cmd get_registry.Query) (get_registry.Registry, error) {
	return builder.
		Query[get_registry.Registry](`
		SELECT
			registry.id
			,registry.name
			,registry.url
			,registry.credentials_username
			,registry.credentials_password
			,registry.created_at
			,creator.id
			,creator.email
		FROM [deployment.registries] registry
		INNER JOIN [auth.users] creator ON creator.id = registry.created_by
		WHERE registry.id = ?`, cmd.ID).
		One(s.db, ctx, registryMapper)
}

var getDeploymentDataloader = builder.NewDataloader(
	func(a get_apps.App) string { return a.ID },
	func(e builder.Executor, ctx context.Context, kr storage.KeyedResult[get_apps.App]) error {
		_, err := builder.
			Query[get_app_deployments.Deployment](`
			SELECT
				deployment.app_id
				,deployment.deployment_number
				,deployment.config_environment
				,deployment.config_target
				,target.name
				,target.url
				,target.state_status
				,deployment.source_discriminator
				,deployment.source
				,deployment.state_status
				,deployment.state_errcode
				,deployment.state_started_at
				,deployment.state_finished_at
				,deployment.requested_at
				,requester.id
				,requester.email
				,MAX(deployment.requested_at) AS max_requested_at
			FROM [deployment.deployments] deployment
			INNER JOIN [auth.users] requester ON requester.id = deployment.requested_by
			LEFT JOIN [deployment.targets] target ON target.id = deployment.config_target`).
			S(builder.Array("WHERE deployment.app_id IN", kr.Keys())).
			F("GROUP BY deployment.app_id, deployment.config_environment").
			All(e, ctx, deploymentMapper(kr))

		return err
	})

var getDeploymentDetailDataloader = builder.NewDataloader(
	func(a get_app_detail.App) string { return a.ID },
	func(e builder.Executor, ctx context.Context, kr storage.KeyedResult[get_app_detail.App]) error {
		_, err := builder.
			Query[get_deployment.Deployment](`
			SELECT
				deployment.app_id
				,deployment.deployment_number
				,deployment.config_environment
				,deployment.config_target
				,target.name
				,target.url
				,target.state_status
				,target.entrypoints
				,deployment.source_discriminator
				,deployment.source
				,deployment.state_status
				,deployment.state_errcode
				,deployment.state_services
				,deployment.state_started_at
				,deployment.state_finished_at
				,deployment.requested_at
				,requester.id
				,requester.email
				,MAX(deployment.requested_at) AS max_requested_at
			FROM [deployment.deployments] deployment
			INNER JOIN [auth.users] requester ON requester.id = deployment.requested_by
			LEFT JOIN [deployment.targets] target ON target.id = deployment.config_target`).
			S(builder.Array("WHERE deployment.app_id IN", kr.Keys())).
			F("GROUP BY deployment.app_id, deployment.config_environment").
			All(e, ctx, deploymentDetailMapper(kr))

		return err
	})

// AppData scanner which include last deployments by environment.
func appDataMapper(s storage.Scanner) (a get_apps.App, err error) {
	var (
		cleanupRequestedById    monad.Maybe[string]
		cleanupRequestedByEmail monad.Maybe[string]
	)

	err = s.Scan(
		&a.ID,
		&a.Name,
		&a.CleanupRequestedAt,
		&cleanupRequestedById,
		&cleanupRequestedByEmail,
		&a.CreatedAt,
		&a.CreatedBy.ID,
		&a.CreatedBy.Email,
		&a.ProductionTarget.ID,
		&a.ProductionTarget.Name,
		&a.ProductionTarget.Url,
		&a.StagingTarget.ID,
		&a.StagingTarget.Name,
		&a.StagingTarget.Url,
	)

	if id, isSet := cleanupRequestedById.TryGet(); isSet {
		a.CleanupRequestedBy.Set(app.UserSummary{
			ID:    id,
			Email: cleanupRequestedByEmail.MustGet(),
		})
	}

	return a, err
}

// Same as the appDataMapper but includes the app's environment variables.
func appDetailDataMapper(s storage.Scanner) (a get_app_detail.App, err error) {
	var (
		url                     monad.Maybe[string]
		token                   monad.Maybe[storage.SecretString]
		cleanupRequestedById    monad.Maybe[string]
		cleanupRequestedByEmail monad.Maybe[string]

		productionMigrationId   monad.Maybe[string]
		productionMigrationName monad.Maybe[string]
		productionMigrationUrl  monad.Maybe[string]

		stagingMigrationId   monad.Maybe[string]
		stagingMigrationName monad.Maybe[string]
		stagingMigrationUrl  monad.Maybe[string]
	)

	if err = s.Scan(
		&a.ID,
		&a.Name,
		&url,
		&token,
		&a.Production.Target.ID,
		&a.Production.Target.Name,
		&a.Production.Target.Url,
		&productionMigrationId,
		&productionMigrationName,
		&productionMigrationUrl,
		&a.Production.Vars,
		&a.Staging.Target.ID,
		&a.Staging.Target.Name,
		&a.Staging.Target.Url,
		&stagingMigrationId,
		&stagingMigrationName,
		&stagingMigrationUrl,
		&a.Staging.Vars,
		&a.CleanupRequestedAt,
		&cleanupRequestedById,
		&cleanupRequestedByEmail,
		&a.CreatedAt,
		&a.CreatedBy.ID,
		&a.CreatedBy.Email,
	); err != nil {
		return a, err
	}

	if productionMigrationTarget, isSet := productionMigrationId.TryGet(); isSet {
		a.Production.Migration.Set(app.TargetSummary{
			ID:   productionMigrationTarget,
			Name: productionMigrationName.MustGet(),
			Url:  productionMigrationUrl,
		})
	}

	if stagingMigrationTarget, isSet := stagingMigrationId.TryGet(); isSet {
		a.Staging.Migration.Set(app.TargetSummary{
			ID:   stagingMigrationTarget,
			Name: stagingMigrationName.MustGet(),
			Url:  stagingMigrationUrl,
		})
	}

	if u, isSet := url.TryGet(); isSet {
		a.VersionControl.Set(get_app_detail.VersionControl{
			Url:   u,
			Token: token,
		})
	}

	if id, isSet := cleanupRequestedById.TryGet(); isSet {
		a.CleanupRequestedBy.Set(app.UserSummary{
			ID:    id,
			Email: cleanupRequestedByEmail.MustGet(),
		})
	}

	return a, err
}

func deploymentMapper(kr storage.KeyedResult[get_apps.App]) storage.Mapper[get_app_deployments.Deployment] {
	return func(scanner storage.Scanner) (d get_app_deployments.Deployment, err error) {
		var (
			maxRequestedAt string
			sourceData     string
			targetStatus   *uint8
		)

		if err = scanner.Scan(
			&d.AppID,
			&d.DeploymentNumber,
			&d.Environment,
			&d.Target.ID,
			&d.Target.Name,
			&d.Target.Url,
			&targetStatus,
			&d.Source.Discriminator,
			&sourceData,
			&d.State.Status,
			&d.State.ErrCode,
			&d.State.StartedAt,
			&d.State.FinishedAt,
			&d.RequestedAt,
			&d.RequestedBy.ID,
			&d.RequestedBy.Email,
			&maxRequestedAt,
		); err != nil {
			return d, err
		}

		// Can't scan directly into a monad.Maybe or it will fail with a conversion error between int64/uint8
		if targetStatus != nil {
			d.Target.Status.Set(*targetStatus)
		}

		d.Source.Data, err = get_deployment.SourceDataTypes.From(d.Source.Discriminator, sourceData)

		if kr != nil {
			kr.Update(d.AppID, func(a get_apps.App) get_apps.App {
				switch domain.EnvironmentName(d.Environment) {
				case domain.Production:
					a.LatestDeployments.Production.Set(d)
				case domain.Staging:
					a.LatestDeployments.Staging.Set(d)
				}
				return a
			})
		}

		return d, err
	}
}

func deploymentDetailMapper(kr storage.KeyedResult[get_app_detail.App]) storage.Mapper[get_deployment.Deployment] {
	return func(scanner storage.Scanner) (d get_deployment.Deployment, err error) {
		var (
			maxRequestedAt string
			sourceData     string
			targetStatus   *uint8
		)

		if err = scanner.Scan(
			&d.AppID,
			&d.DeploymentNumber,
			&d.Environment,
			&d.Target.ID,
			&d.Target.Name,
			&d.Target.Url,
			&targetStatus,
			&d.Target.Entrypoints,
			&d.Source.Discriminator,
			&sourceData,
			&d.State.Status,
			&d.State.ErrCode,
			&d.State.Services,
			&d.State.StartedAt,
			&d.State.FinishedAt,
			&d.RequestedAt,
			&d.RequestedBy.ID,
			&d.RequestedBy.Email,
			&maxRequestedAt,
		); err != nil {
			return d, err
		}

		// Can't scan directly into a monad.Maybe or it will fail with a conversion error between int64/uint8
		if targetStatus != nil {
			d.Target.Status.Set(*targetStatus)
		}

		d.Source.Data, err = get_deployment.SourceDataTypes.From(d.Source.Discriminator, sourceData)

		d.ResolveServicesUrls()

		if kr != nil {
			kr.Update(d.AppID, func(a get_app_detail.App) get_app_detail.App {
				switch domain.EnvironmentName(d.Environment) {
				case domain.Production:
					a.LatestDeployments.Production.Set(d)
				case domain.Staging:
					a.LatestDeployments.Staging.Set(d)
				}
				return a
			})
		}

		return d, err
	}
}

func targetMapper(scanner storage.Scanner) (t get_target.Target, err error) {
	var (
		providerData            string
		cleanupRequestedById    monad.Maybe[string]
		cleanupRequestedByEmail monad.Maybe[string]
	)

	if err = scanner.Scan(
		&t.ID,
		&t.Name,
		&t.Url,
		&t.Provider.Kind,
		&providerData,
		&t.State.Status,
		&t.State.ErrCode,
		&t.State.LastReadyVersion,
		&t.CleanupRequestedAt,
		&cleanupRequestedById,
		&cleanupRequestedByEmail,
		&t.CreatedAt,
		&t.CreatedBy.ID,
		&t.CreatedBy.Email,
	); err != nil {
		return t, err
	}

	if id, isSet := cleanupRequestedById.TryGet(); isSet {
		t.CleanupRequestedBy.Set(app.UserSummary{
			ID:    id,
			Email: cleanupRequestedByEmail.MustGet(),
		})
	}

	t.Provider.Data, err = get_target.ProviderConfigTypes.From(t.Provider.Kind, providerData)

	return t, err
}

func registryMapper(scanner storage.Scanner) (r get_registry.Registry, err error) {
	var (
		credentialsUsername monad.Maybe[string]
		credentialsPassword monad.Maybe[storage.SecretString]
	)

	if err = scanner.Scan(
		&r.ID,
		&r.Name,
		&r.Url,
		&credentialsUsername,
		&credentialsPassword,
		&r.CreatedAt,
		&r.CreatedBy.ID,
		&r.CreatedBy.Email,
	); err != nil {
		return r, err
	}

	if usr, isSet := credentialsUsername.TryGet(); isSet {
		r.Credentials.Set(get_registry.Credentials{
			Username: usr,
			Password: credentialsPassword.Get(""),
		})
	}

	return r, err
}
