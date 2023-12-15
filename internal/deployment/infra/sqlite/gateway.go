package sqlite

import (
	"context"

	"github.com/YuukanOO/seelf/internal/deployment/app/get_app_deployments"
	"github.com/YuukanOO/seelf/internal/deployment/app/get_app_detail"
	"github.com/YuukanOO/seelf/internal/deployment/app/get_apps"
	"github.com/YuukanOO/seelf/internal/deployment/app/get_deployment"
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
				,apps.created_at
				,users.id
				,users.email
			FROM apps
			INNER JOIN users ON users.id = apps.created_by`).
		All(s.db, ctx, appDataMapper, appLastDeploymentsByEnvDataloader)
}

func (s *gateway) GetAppByID(ctx context.Context, cmd get_app_detail.Query) (get_app_detail.App, error) {
	return builder.
		Query[get_app_detail.App](`
			SELECT
				apps.id
				,apps.name
				,apps.vcs_url
				,apps.vcs_token
				,apps.env
				,apps.cleanup_requested_at
				,apps.created_at
				,users.id
				,users.email
			FROM apps
			INNER JOIN users ON users.id = apps.created_by
			WHERE apps.id = ?`, cmd.ID).
		One(s.db, ctx, appDetailDataMapper, appDetailLastDeploymentsByEnvDataloader)
}

func (s *gateway) GetAllDeploymentsByApp(ctx context.Context, cmd get_app_deployments.Query) (storage.Paginated[get_deployment.Deployment], error) {
	return builder.
		Select[get_deployment.Deployment](`
			deployments.app_id
			,deployments.deployment_number
			,deployments.config_environment
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
			WHERE deployments.app_id = ?`, cmd.AppID).
		S(builder.MaybeValue(cmd.Environment, "AND deployments.config_environment = ?")).
		F("ORDER BY deployments.deployment_number DESC").
		Paginate(s.db, ctx, deploymentMapper, cmd.Page.Get(1))
}

func (s *gateway) GetDeploymentByID(ctx context.Context, cmd get_deployment.Query) (get_deployment.Deployment, error) {
	return builder.
		Query[get_deployment.Deployment](`
		SELECT
			deployments.app_id
			,deployments.deployment_number
			,deployments.config_environment
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
		WHERE deployments.app_id = ? AND deployments.deployment_number = ?`, cmd.AppID, cmd.DeploymentNumber).
		One(s.db, ctx, deploymentMapper)
}

// Specific case because the deployments dataloader can be use to fill the App and AppDetail
// structs. So this function will be build the appropriate dataloader for each case.
func newAppWithLastDeploymentsByEnvDataloader[T any](
	extractor func(T) string,
	merger storage.Merger[T, get_deployment.Deployment],
) builder.Dataloader[T] {
	return builder.NewDataloader[T](
		extractor,
		func(e builder.Executor, ctx context.Context, kr builder.KeyedResult[T]) error {
			_, err := builder.
				Query[get_deployment.Deployment](`
			SELECT
				deployments.app_id -- The first one will be used by the dataloader merge process
				,deployments.app_id
				,deployments.deployment_number
				,deployments.config_environment
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
			INNER JOIN users ON users.id = deployments.requested_by`).
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
		a.Environments[d.Environment] = d
		return a
	},
)

var appDetailLastDeploymentsByEnvDataloader = newAppWithLastDeploymentsByEnvDataloader(
	func(a get_app_detail.App) string { return a.ID },
	func(a get_app_detail.App, d get_deployment.Deployment) get_app_detail.App {
		a.Environments[d.Environment] = d
		return a
	},
)

// AppData scanner which include last deployments by environment.
func appDataMapper(s storage.Scanner) (a get_apps.App, err error) {
	err = s.Scan(
		&a.ID,
		&a.Name,
		&a.CleanupRequestedAt,
		&a.CreatedAt,
		&a.CreatedBy.ID,
		&a.CreatedBy.Email,
	)

	a.Environments = make(map[string]get_deployment.Deployment)

	return a, err
}

// Same as the appDataMapper but includes the app's environment variables.
func appDetailDataMapper(s storage.Scanner) (a get_app_detail.App, err error) {
	var (
		url   monad.Maybe[string]
		token monad.Maybe[storage.SecretString]
	)

	err = s.Scan(
		&a.ID,
		&a.Name,
		&url,
		&token,
		&a.Env,
		&a.CleanupRequestedAt,
		&a.CreatedAt,
		&a.CreatedBy.ID,
		&a.CreatedBy.Email,
	)

	a.Environments = make(map[string]get_deployment.Deployment)

	if u, isSet := url.TryGet(); isSet {
		a.VCS = a.VCS.WithValue(get_app_detail.VCSConfig{
			Url:   u,
			Token: token,
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

	return d, err
}

func deploymentMapper(scanner storage.Scanner) (d get_deployment.Deployment, err error) {
	var sourceData string

	err = scanner.Scan(
		&d.AppID,
		&d.DeploymentNumber,
		&d.Environment,
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

	return d, err
}
