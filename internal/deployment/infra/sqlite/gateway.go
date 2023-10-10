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
		QueryEx[query.App, appDataScanner[query.App]](`
			SELECT
				apps.id
				,apps.name
				,apps.cleanup_requested_at
				,apps.created_at
				,users.id
				,users.email
			FROM apps
			INNER JOIN users ON users.id = apps.created_by`).
		AllEx(s, ctx, appDataScannerBuilder[query.App], appDataMapper)
}

func (s *gateway) GetAppByID(ctx context.Context, appid string) (query.AppDetail, error) {
	return builder.
		QueryEx[query.AppDetail, appDataScanner[query.AppDetail]](`
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
		OneEx(s, ctx, appDataScannerBuilder[query.AppDetail], appDetailDataMapper)
}

func (s *gateway) GetAllDeploymentsByApp(ctx context.Context, appid string, filters query.GetDeploymentsFilters) (shared.Paginated[query.Deployment], error) {
	return builder.
		Select[query.Deployment](`
			deployments.app_id
			,deployments.deployment_number
			,deployments.config_environment
			,deployments.source_kind
			,deployments.source_data
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
			,deployments.source_kind
			,deployments.source_data
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

func (s *gateway) GetDeploymentLogfileByID(ctx context.Context, appid string, deploymentNumber int) (string, error) {
	return builder.
		Query[string]("SELECT state_logfile FROM deployments WHERE app_id = ? AND deployment_number = ?", appid, deploymentNumber).
		Extract(s, ctx)
}

type (
	appDataScanner[T any] interface {
		storage.Scanner
		Deployments(storage.KeyedMapper[query.Deployment, storage.Scanner], storage.Merger[T, query.Deployment])
	}

	concreteAppDataScanner[T any] struct {
		conn             builder.Executor
		row              storage.Scanner
		deploymentMapper storage.KeyedMapper[query.Deployment, storage.Scanner]
		deploymentMerger storage.Merger[T, query.Deployment]
	}
)

func appDataScannerBuilder[T any](conn builder.Executor) builder.ScannerEx[T, appDataScanner[T]] {
	return &concreteAppDataScanner[T]{
		conn: conn,
	}
}

func (s *concreteAppDataScanner[T]) Scan(dest ...any) error {
	return s.row.Scan(dest...)
}

func (s *concreteAppDataScanner[T]) Contextualize(row storage.Scanner) appDataScanner[T] {
	s.row = row
	return s
}

func (s *concreteAppDataScanner[T]) Finalize(ctx context.Context, results builder.KeyedResult[T]) ([]T, error) {
	_, err := builder.
		Query[query.Deployment](`
			SELECT
				deployments.app_id
				,deployments.deployment_number
				,deployments.config_environment
				,deployments.source_kind
				,deployments.source_data
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
		S(builder.Array("WHERE deployments.app_id IN", results.Keys())).
		F("GROUP BY deployments.app_id, deployments.config_environment").
		All(s.conn, ctx, builder.Merge(results, s.deploymentMapper, s.deploymentMerger))

	if err != nil {
		return nil, err
	}

	return results.Data(), nil
}

func (s *concreteAppDataScanner[T]) Deployments(
	mapper storage.KeyedMapper[query.Deployment, storage.Scanner],
	merger storage.Merger[T, query.Deployment],
) {
	s.deploymentMapper = mapper
	s.deploymentMerger = merger
}

// AppData scanner which include last deployments by environment.
func appDataMapper(s appDataScanner[query.App]) (_ string, a query.App, err error) {
	err = s.Scan(
		&a.ID,
		&a.Name,
		&a.CleanupRequestedAt,
		&a.CreatedAt,
		&a.CreatedBy.ID,
		&a.CreatedBy.Email,
	)

	a.Environments = make(map[string]query.Deployment)

	s.Deployments(keyedDeploymentScanner, func(app query.App, depl query.Deployment) query.App {
		app.Environments[depl.Environment] = depl
		return app
	},
	)

	return a.ID, a, err
}

// Same as the appDataMapper but includes the app's environment variables.
func appDetailDataMapper(s appDataScanner[query.AppDetail]) (_ string, a query.AppDetail, err error) {
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

	s.Deployments(keyedDeploymentScanner, func(app query.AppDetail, depl query.Deployment) query.AppDetail {
		app.Environments[depl.Environment] = depl
		return app
	},
	)

	return a.ID, a, err
}

func keyedDeploymentScanner(s storage.Scanner) (key string, d query.Deployment, err error) {
	var maxRequestedAt string
	err = s.Scan(
		&d.AppID,
		&d.DeploymentNumber,
		&d.Environment,
		&d.Meta.Kind,
		&d.Meta.Data,
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

	return d.AppID, d, err
}

func deploymentMapper(scanner storage.Scanner) (d query.Deployment, err error) {
	err = scanner.Scan(
		&d.AppID,
		&d.DeploymentNumber,
		&d.Environment,
		&d.Meta.Kind,
		&d.Meta.Data,
		&d.State.Status,
		&d.State.ErrCode,
		&d.State.Services,
		&d.State.StartedAt,
		&d.State.FinishedAt,
		&d.RequestedAt,
		&d.RequestedBy.ID,
		&d.RequestedBy.Email,
	)

	return d, err
}
