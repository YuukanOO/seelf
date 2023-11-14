package sqlite

import (
	"context"

	"github.com/YuukanOO/seelf/internal/deployment/app/query"
	"github.com/YuukanOO/seelf/pkg/monad"
	shared "github.com/YuukanOO/seelf/pkg/query"
	"github.com/YuukanOO/seelf/pkg/storage"
	"github.com/YuukanOO/seelf/pkg/storage/sqlite"
	"github.com/YuukanOO/seelf/pkg/storage/sqlite/builder"
)

type gateway struct {
	sqlite.Database
}

func NewGateway(db sqlite.Database) query.Gateway {
	return &gateway{db}
}

func (s *gateway) GetAllApps(ctx context.Context) ([]query.App, error) {
	return builder.
		Query[query.App](`
			SELECT
				apps.id
				,apps.name
				,apps.cleanup_requested_at
				,apps.created_at
				,users.id
				,users.email
			FROM apps
			INNER JOIN users ON users.id = apps.created_by`).
		All(s, ctx, appDataMapper, appLastDeploymentsByEnvDataloader)
}

func (s *gateway) GetAppByID(ctx context.Context, appid string) (query.AppDetail, error) {
	return builder.
		Query[query.AppDetail](`
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
			WHERE apps.id = ?`, appid).
		One(s, ctx, appDetailDataMapper, appDetailLastDeploymentsByEnvDataloader)
}

func (s *gateway) GetAllDeploymentsByApp(ctx context.Context, appid string, filters query.GetDeploymentsFilters) (shared.Paginated[query.Deployment], error) {
	return builder.
		Select[query.Deployment](`
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
			WHERE deployments.app_id = ?`, appid).
		S(builder.MaybeValue(filters.Environment, "AND deployments.config_environment = ?")).
		F("ORDER BY deployments.deployment_number DESC").
		Paginate(s, ctx, deploymentMapper, filters.Page.Get(1))
}

func (s *gateway) GetDeploymentByID(ctx context.Context, appid string, deploymentNumber int) (query.Deployment, error) {
	return builder.
		Query[query.Deployment](`
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
		WHERE deployments.app_id = ? AND deployments.deployment_number = ?`, appid, deploymentNumber).
		One(s, ctx, deploymentMapper)
}

// Specific case because the deployments dataloader can be use to fill the App and AppDetail
// structs. So this function will be build the appropriate dataloader for each case.
func newAppWithLastDeploymentsByEnvDataloader[T any](
	extractor func(T) string,
	merger storage.Merger[T, query.Deployment],
) builder.Dataloader[T] {
	return builder.NewDataloader[T](
		extractor,
		func(e builder.Executor, ctx context.Context, kr builder.KeyedResult[T]) error {
			_, err := builder.
				Query[query.Deployment](`
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
	func(a query.App) string { return a.ID },
	func(a query.App, d query.Deployment) query.App {
		a.Environments[d.Environment] = d
		return a
	},
)

var appDetailLastDeploymentsByEnvDataloader = newAppWithLastDeploymentsByEnvDataloader(
	func(a query.AppDetail) string { return a.ID },
	func(a query.AppDetail, d query.Deployment) query.AppDetail {
		a.Environments[d.Environment] = d
		return a
	},
)

// AppData scanner which include last deployments by environment.
func appDataMapper(s storage.Scanner) (a query.App, err error) {
	err = s.Scan(
		&a.ID,
		&a.Name,
		&a.CleanupRequestedAt,
		&a.CreatedAt,
		&a.CreatedBy.ID,
		&a.CreatedBy.Email,
	)

	a.Environments = make(map[string]query.Deployment)

	return a, err
}

// Same as the appDataMapper but includes the app's environment variables.
func appDetailDataMapper(s storage.Scanner) (a query.AppDetail, err error) {
	var (
		url   monad.Maybe[string]
		token monad.Maybe[shared.SecretString]
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

	a.Environments = make(map[string]query.Deployment)

	if url.HasValue() {
		a.VCS = a.VCS.WithValue(query.VCSConfig{
			Url:   url.MustGet(),
			Token: token,
		})
	}

	return a, err
}

func lastDeploymentMapper(s storage.Scanner) (d query.Deployment, err error) {
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

	d.Source.Data, err = query.SourceDataTypes.From(d.Source.Discriminator, sourceData)

	return d, err
}

func deploymentMapper(scanner storage.Scanner) (d query.Deployment, err error) {
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

	d.Source.Data, err = query.SourceDataTypes.From(d.Source.Discriminator, sourceData)

	return d, err
}
