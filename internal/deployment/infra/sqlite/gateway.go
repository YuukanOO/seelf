package sqlite

import (
	"context"
	"strings"

	"github.com/YuukanOO/seelf/internal/deployment/app"
	"github.com/YuukanOO/seelf/internal/deployment/app/get_app_deployments"
	"github.com/YuukanOO/seelf/internal/deployment/app/get_app_detail"
	"github.com/YuukanOO/seelf/internal/deployment/app/get_apps"
	"github.com/YuukanOO/seelf/internal/deployment/app/get_deployment"
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
		All(s.db, ctx, appDataMapper, appLastDeploymentsByEnvDataloader)
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
		One(s.db, ctx, appDetailDataMapper, appDetailLastDeploymentsByEnvDataloader)
}

func (s *gateway) GetAllDeploymentsByApp(ctx context.Context, cmd get_app_deployments.Query) (storage.Paginated[get_deployment.Deployment], error) {
	return builder.
		Select[get_deployment.Deployment](`
			deployments.app_id
			,deployments.deployment_number
			,deployments.config_environment
			,deployments.config_target
			,targets.name
			,targets.url
			,deployments.source_discriminator
			,deployments.source
			,deployments.state_status
			,deployments.state_errcode
			,deployments.state_services
			,deployments.state_started_at
			,deployments.state_finished_at
			,deployments.requested_at
			,users.id
			,users.email`).
		F(`
			FROM deployments
			INNER JOIN users ON users.id = deployments.requested_by
			LEFT JOIN targets ON targets.id = deployments.config_target
			WHERE deployments.app_id = ?`, cmd.AppID).
		S(builder.MaybeValue(cmd.Environment, "AND deployments.config_environment = ?")).
		F("ORDER BY deployments.deployment_number DESC").
		Paginate(s.db, ctx, deploymentMapper, cmd.Page.Get(1), 5)
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
		FROM deployments
		INNER JOIN users ON users.id = deployments.requested_by
		LEFT JOIN targets ON targets.id = deployments.config_target
		WHERE deployments.app_id = ? AND deployments.deployment_number = ?`, cmd.AppID, cmd.DeploymentNumber).
		One(s.db, ctx, deploymentMapper)
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

// Specific case because the deployments dataloader can be use to fill the App and AppDetail
// structs. So this function will be build the appropriate dataloader for each case.
func newAppWithLastDeploymentsByEnvDataloader[T any](
	extractor func(T) string,
	merger storage.Merger[T, get_deployment.Deployment],
) builder.Dataloader[T] {
	return builder.NewDataloader(
		extractor,
		func(e builder.Executor, ctx context.Context, kr builder.KeyedResult[T]) error {
			_, err := builder.
				Query[get_deployment.Deployment](`
			SELECT
				deployments.app_id -- The first one will be used by the dataloader merge process
				,deployments.app_id
				,deployments.deployment_number
				,deployments.config_environment
				,deployments.config_target
				,targets.name
				,targets.url
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
				All(e, ctx, builder.Merge(kr, lastDeploymentMapper, merger))

			return err
		},
	)
}

var appLastDeploymentsByEnvDataloader = newAppWithLastDeploymentsByEnvDataloader(
	func(a get_apps.App) string { return a.ID },
	func(a get_apps.App, d get_deployment.Deployment) get_apps.App {
		switch domain.Environment(d.Environment) {
		case domain.Production:
			a.LatestDeployments.Production.Set(d)
		case domain.Staging:
			a.LatestDeployments.Staging.Set(d)
		}
		return a
	},
)

var appDetailLastDeploymentsByEnvDataloader = newAppWithLastDeploymentsByEnvDataloader(
	func(a get_app_detail.App) string { return a.ID },
	func(a get_app_detail.App, d get_deployment.Deployment) get_app_detail.App {
		switch domain.Environment(d.Environment) {
		case domain.Production:
			a.LatestDeployments.Production.Set(d)
		case domain.Staging:
			a.LatestDeployments.Staging.Set(d)
		}
		return a
	},
)

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

func lastDeploymentMapper(s storage.Scanner) (d get_deployment.Deployment, err error) {
	var (
		maxRequestedAt string
		sourceData     string
	)

	err = s.Scan(
		&d.AppID,
		&d.DeploymentNumber,
		&d.Environment,
		&d.Target.ID,
		&d.Target.Name,
		&d.Target.Url,
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
		&maxRequestedAt, // Needed because go-sqlite3 lib could not extract max(requested_at) into a time.Time... I may switch to another lib in the future
	)

	if err != nil {
		return d, err
	}

	d.Source.Data, err = get_deployment.SourceDataTypes.From(d.Source.Discriminator, sourceData)

	populateServicesUrls(d.Target.Url, d.State.Services)

	return d, err
}

func deploymentMapper(scanner storage.Scanner) (d get_deployment.Deployment, err error) {
	var sourceData string

	err = scanner.Scan(
		&d.AppID,
		&d.DeploymentNumber,
		&d.Environment,
		&d.Target.ID,
		&d.Target.Name,
		&d.Target.Url,
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
	)

	if err != nil {
		return d, err
	}

	d.Source.Data, err = get_deployment.SourceDataTypes.From(d.Source.Discriminator, sourceData)

	populateServicesUrls(d.Target.Url, d.State.Services)

	return d, err
}

// Since the target domain is dynamic, compute exposed service urls based on the presence
// of the given current target url.
func populateServicesUrls(targetUrl monad.Maybe[string], services monad.Maybe[get_deployment.Services]) {
	ss, hasServices := services.TryGet()
	url, hasUrl := targetUrl.TryGet()

	if !hasUrl || !hasServices {
		return
	}

	idx := strings.Index(url, "://")
	scheme, host := url[:idx+3], url[idx+3:]

	for i, s := range ss {
		subdomain, isExposed := s.Subdomain.TryGet()

		if !isExposed {
			continue
		}

		s.Url.Set(scheme + subdomain + "." + host)

		ss[i] = s
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
