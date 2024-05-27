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

type gateway struct {
	db *sqlite.Database
}

func NewGateway(db *sqlite.Database) *gateway {
	return &gateway{db}
}

func (s *gateway) GetAllApps(ctx context.Context, cmd get_apps.Query) ([]get_apps.App, error) {
	return builder.
		Query[get_apps.App](`
			SELECT
				apps.id
				,apps.name
				,apps.cleanup_requested_at
				,cusers.id
				,cusers.email
				,apps.created_at
				,users.id
				,users.email
				,production_target.id
				,production_target.name
				,production_target.url
				,staging_target.id
				,staging_target.name
				,staging_target.url
			FROM apps
			INNER JOIN users ON users.id = apps.created_by
			INNER JOIN targets AS production_target ON production_target.id = apps.production_target
			INNER JOIN targets AS staging_target ON staging_target.id = apps.staging_target
			LEFT JOIN users cusers ON cusers.id = apps.cleanup_requested_by`).
		All(s.db, ctx, appDataMapper, getDeploymentDataloader)
}

func (s *gateway) GetAppByID(ctx context.Context, cmd get_app_detail.Query) (get_app_detail.App, error) {
	return builder.
		Query[get_app_detail.App](`
			SELECT
				apps.id
				,apps.name
				,apps.version_control_url
				,apps.version_control_token
				,production_target.id
				,production_target.name
				,production_target.url
				,apps.production_vars
				,staging_target.id
				,staging_target.name
				,staging_target.url
				,apps.staging_vars
				,apps.cleanup_requested_at
				,cusers.id
				,cusers.email
				,apps.created_at
				,users.id
				,users.email
			FROM apps
			INNER JOIN users ON users.id = apps.created_by
			INNER JOIN targets production_target ON production_target.id = apps.production_target
			INNER JOIN targets staging_target ON staging_target.id = apps.staging_target
			LEFT JOIN users cusers ON cusers.id = apps.cleanup_requested_by
			WHERE apps.id = ?`, cmd.ID).
		One(s.db, ctx, appDetailDataMapper, getDeploymentDetailDataloader)
}

func (s *gateway) GetAllDeploymentsByApp(ctx context.Context, cmd get_app_deployments.Query) (storage.Paginated[get_app_deployments.Deployment], error) {
	return builder.
		Select[get_app_deployments.Deployment](`
			deployments.app_id
			,deployments.deployment_number
			,deployments.config_environment
			,deployments.config_target
			,targets.name
			,targets.url
			,targets.state_status
			,deployments.source_discriminator
			,deployments.source
			,deployments.state_status
			,deployments.state_errcode
			,deployments.state_started_at
			,deployments.state_finished_at
			,deployments.requested_at
			,users.id
			,users.email
			,'' -- only to use the same mapper as the latest deployments`).
		F(`
			FROM deployments
			INNER JOIN users ON users.id = deployments.requested_by
			LEFT JOIN targets ON targets.id = deployments.config_target
			WHERE deployments.app_id = ?`, cmd.AppID).
		S(builder.MaybeValue(cmd.Environment, "AND deployments.config_environment = ?")).
		F("ORDER BY deployments.deployment_number DESC").
		Paginate(s.db, ctx, deploymentMapper(nil), cmd.Page.Get(1), 5)
}

func (s *gateway) GetDeploymentByID(ctx context.Context, cmd get_deployment.Query) (get_deployment.Deployment, error) {
	return builder.
		Query[get_deployment.Deployment](`
		SELECT
			deployments.app_id
			,deployments.deployment_number
			,deployments.config_environment
			,deployments.config_target
			,targets.name
			,targets.url
			,targets.state_status
			,targets.entrypoints
			,deployments.source_discriminator
			,deployments.source
			,deployments.state_status
			,deployments.state_errcode
			,deployments.state_services
			,deployments.state_started_at
			,deployments.state_finished_at
			,deployments.requested_at
			,users.id
			,users.email
			,'' -- only to use the same mapper as the latest deployments
		FROM deployments
		INNER JOIN users ON users.id = deployments.requested_by
		LEFT JOIN targets ON targets.id = deployments.config_target
		WHERE deployments.app_id = ? AND deployments.deployment_number = ?`, cmd.AppID, cmd.DeploymentNumber).
		One(s.db, ctx, deploymentDetailMapper(nil))
}

func (s *gateway) GetAllTargets(ctx context.Context, cmd get_targets.Query) ([]get_target.Target, error) {
	return builder.
		Query[get_target.Target](`
		SELECT
			targets.id
			,targets.name
			,targets.url
			,targets.provider_kind
			,targets.provider
			,targets.state_status
			,targets.state_errcode
			,targets.state_last_ready_version
			,targets.cleanup_requested_at
			,cusers.id
			,cusers.email
			,targets.created_at
			,users.id
			,users.email
		FROM targets
		INNER JOIN users ON users.id = targets.created_by
		LEFT JOIN users cusers ON cusers.id = targets.cleanup_requested_by
		WHERE TRUE
		`).
		S(builder.If(cmd.ActiveOnly, "AND targets.cleanup_requested_at IS NULL")).
		All(s.db, ctx, targetMapper)
}

func (s *gateway) GetTargetByID(ctx context.Context, cmd get_target.Query) (get_target.Target, error) {
	return builder.
		Query[get_target.Target](`
		SELECT
			targets.id
			,targets.name
			,targets.url
			,targets.provider_kind
			,targets.provider
			,targets.state_status
			,targets.state_errcode
			,targets.state_last_ready_version
			,targets.cleanup_requested_at
			,cusers.id
			,cusers.email
			,targets.created_at
			,users.id
			,users.email
		FROM targets
		INNER JOIN users ON users.id = targets.created_by
		LEFT JOIN users cusers ON cusers.id = targets.cleanup_requested_by
		WHERE targets.id = ?`, cmd.ID).
		One(s.db, ctx, targetMapper)
}

func (s *gateway) GetRegistries(ctx context.Context, cmd get_registries.Query) ([]get_registry.Registry, error) {
	return builder.
		Query[get_registry.Registry](`
		SELECT
			registries.id
			,registries.name
			,registries.url
			,registries.credentials_username
			,registries.credentials_password
			,registries.created_at
			,users.id
			,users.email
		FROM registries
		INNER JOIN users ON users.id = registries.created_by`).
		All(s.db, ctx, registryMapper)
}

func (s *gateway) GetRegistryByID(ctx context.Context, cmd get_registry.Query) (get_registry.Registry, error) {
	return builder.
		Query[get_registry.Registry](`
		SELECT
			registries.id
			,registries.name
			,registries.url
			,registries.credentials_username
			,registries.credentials_password
			,registries.created_at
			,users.id
			,users.email
		FROM registries
		INNER JOIN users ON users.id = registries.created_by
		WHERE registries.id = ?`, cmd.ID).
		One(s.db, ctx, registryMapper)
}

var getDeploymentDataloader = builder.NewDataloader(
	func(a get_apps.App) string { return a.ID },
	func(e builder.Executor, ctx context.Context, kr storage.KeyedResult[get_apps.App]) error {
		_, err := builder.
			Query[get_app_deployments.Deployment](`
			SELECT
				deployments.app_id
				,deployments.deployment_number
				,deployments.config_environment
				,deployments.config_target
				,targets.name
				,targets.url
				,targets.state_status
				,deployments.source_discriminator
				,deployments.source
				,deployments.state_status
				,deployments.state_errcode
				,deployments.state_started_at
				,deployments.state_finished_at
				,deployments.requested_at
				,users.id
				,users.email
				,MAX(requested_at) AS max_requested_at
			FROM deployments
			INNER JOIN users ON users.id = deployments.requested_by
			LEFT JOIN targets ON targets.id = deployments.config_target`).
			S(builder.Array("WHERE deployments.app_id IN", kr.Keys())).
			F("GROUP BY deployments.app_id, deployments.config_environment").
			All(e, ctx, deploymentMapper(kr))

		return err
	})

var getDeploymentDetailDataloader = builder.NewDataloader(
	func(a get_app_detail.App) string { return a.ID },
	func(e builder.Executor, ctx context.Context, kr storage.KeyedResult[get_app_detail.App]) error {
		_, err := builder.
			Query[get_deployment.Deployment](`
			SELECT
				deployments.app_id
				,deployments.deployment_number
				,deployments.config_environment
				,deployments.config_target
				,targets.name
				,targets.url
				,targets.state_status
				,targets.entrypoints
				,deployments.source_discriminator
				,deployments.source
				,deployments.state_status
				,deployments.state_errcode
				,deployments.state_services
				,deployments.state_started_at
				,deployments.state_finished_at
				,deployments.requested_at
				,users.id
				,users.email
				,MAX(requested_at) AS max_requested_at
			FROM deployments
			INNER JOIN users ON users.id = deployments.requested_by
			LEFT JOIN targets ON targets.id = deployments.config_target`).
			S(builder.Array("WHERE deployments.app_id IN", kr.Keys())).
			F("GROUP BY deployments.app_id, deployments.config_environment").
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
	)

	err = s.Scan(
		&a.ID,
		&a.Name,
		&url,
		&token,
		&a.Production.Target.ID,
		&a.Production.Target.Name,
		&a.Production.Target.Url,
		&a.Production.Vars,
		&a.Staging.Target.ID,
		&a.Staging.Target.Name,
		&a.Staging.Target.Url,
		&a.Staging.Vars,
		&a.CleanupRequestedAt,
		&cleanupRequestedById,
		&cleanupRequestedByEmail,
		&a.CreatedAt,
		&a.CreatedBy.ID,
		&a.CreatedBy.Email,
	)

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

		err = scanner.Scan(
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
		)

		if err != nil {
			return d, err
		}

		// Can't scan directly into a monad.Maybe or it will fail with a conversion error between int64/uint8
		if targetStatus != nil {
			d.Target.Status.Set(*targetStatus)
		}

		d.Source.Data, err = get_deployment.SourceDataTypes.From(d.Source.Discriminator, sourceData)

		if kr != nil {
			kr.Update(d.AppID, func(a get_apps.App) get_apps.App {
				switch domain.Environment(d.Environment) {
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

		err = scanner.Scan(
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
		)

		if err != nil {
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
				switch domain.Environment(d.Environment) {
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

	err = scanner.Scan(
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
	)

	if err != nil {
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

	err = scanner.Scan(
		&r.ID,
		&r.Name,
		&r.Url,
		&credentialsUsername,
		&credentialsPassword,
		&r.CreatedAt,
		&r.CreatedBy.ID,
		&r.CreatedBy.Email,
	)

	if err != nil {
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
