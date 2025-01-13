package sqlite

import (
	"context"
	"time"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	shared "github.com/YuukanOO/seelf/pkg/domain"
	"github.com/YuukanOO/seelf/pkg/event"
	"github.com/YuukanOO/seelf/pkg/storage"
	"github.com/YuukanOO/seelf/pkg/storage/sqlite"
	"github.com/YuukanOO/seelf/pkg/storage/sqlite/builder"
)

type (
	DeploymentsStore interface {
		domain.DeploymentsReader
		domain.DeploymentsWriter
	}

	deploymentsStore struct {
		db *sqlite.Database
	}
)

func NewDeploymentsStore(db *sqlite.Database) DeploymentsStore {
	return &deploymentsStore{db}
}

func (s *deploymentsStore) GetByID(ctx context.Context, id domain.DeploymentID) (domain.Deployment, error) {
	return builder.
		Query[domain.Deployment](`
		SELECT
			app_id
			,deployment_number
			,config_appid
			,config_appname
			,config_environment
			,config_target
			,config_vars
			,state_status
			,state_errcode
			,state_services
			,state_started_at
			,state_finished_at
			,source_discriminator
			,source
			,requested_at
			,requested_by
			,version
		FROM [deployment.deployments]
		WHERE app_id = ? AND deployment_number = ?`, id.AppID(), id.DeploymentNumber()).
		One(s.db, ctx, domain.DeploymentFrom)
}

func (s *deploymentsStore) GetLastDeployment(ctx context.Context, id domain.AppID, env domain.EnvironmentName) (domain.Deployment, error) {
	return builder.
		Query[domain.Deployment](`
		SELECT
			app_id
			,deployment_number
			,config_appid
			,config_appname
			,config_environment
			,config_target
			,config_vars
			,state_status
			,state_errcode
			,state_services
			,state_started_at
			,state_finished_at
			,source_discriminator
			,source
			,requested_at
			,requested_by
			,version
		FROM [deployment.deployments]
		WHERE app_id = ? AND config_environment = ?
		ORDER BY deployment_number DESC
		LIMIT 1`, id, env).
		One(s.db, ctx, domain.DeploymentFrom)
}

func (s *deploymentsStore) GetNextDeploymentNumber(ctx context.Context, appID domain.AppID) (domain.DeploymentNumber, error) {
	// FIXME: find a better way, on postgresql, I could have used a seq to increment the sequence to avoid any potential duplication
	// of a job number but on sqlite, I could not find a way yet.
	c, err := builder.
		Query[uint]("SELECT COUNT(*) FROM [deployment.deployments] WHERE app_id = ?", appID).
		Extract(s.db, ctx)

	if err != nil {
		return 0, err
	}

	return domain.DeploymentNumber(c + 1), nil
}

func (s *deploymentsStore) HasRunningOrPendingDeploymentsOnTarget(ctx context.Context, target domain.TargetID) (domain.HasRunningOrPendingDeploymentsOnTarget, error) {
	r, err := builder.
		Query[bool](`
		SELECT EXISTS(SELECT 1 FROM [deployment.deployments] WHERE config_target = ? AND state_status IN (?, ?))`,
		target, domain.DeploymentStatusRunning, domain.DeploymentStatusPending).
		Extract(s.db, ctx)

	return domain.HasRunningOrPendingDeploymentsOnTarget(r), err
}

func (s *deploymentsStore) HasDeploymentsOnAppTargetEnv(ctx context.Context, app domain.AppID, target domain.TargetID, env domain.EnvironmentName, ti shared.TimeInterval) (
	domain.HasRunningOrPendingDeploymentsOnAppTargetEnv,
	domain.HasSuccessfulDeploymentsOnAppTargetEnv,
	error,
) {
	c, err := builder.
		Query[deploymentsOnAppTargetEnv](`
		SELECT
			EXISTS(
				SELECT 1 FROM [deployment.deployments]
				WHERE 
					app_id = ? AND config_target = ? AND config_environment = ?
					AND state_status IN (?, ?)
			) AS runningOrPending
			,EXISTS(
				SELECT 1 FROM [deployment.deployments]
				WHERE
					app_id = ? AND config_target = ? AND config_environment = ?
					AND state_status = ? AND requested_at >= ? AND requested_at <= ?
			) AS successful`,
		app, target, env, domain.DeploymentStatusPending, domain.DeploymentStatusRunning,
		app, target, env, domain.DeploymentStatusSucceeded, ti.From(), ti.To()).
		One(s.db, ctx, deploymentsOnAppTargetEnvMapper)

	return domain.HasRunningOrPendingDeploymentsOnAppTargetEnv(c.runningOrPending), domain.HasSuccessfulDeploymentsOnAppTargetEnv(c.successful), err
}

func (s *deploymentsStore) FailDeployments(ctx context.Context, reason error, criteria domain.FailCriteria) error {
	now := time.Now().UTC()

	return builder.Update("[deployment.deployments]", builder.Values{
		"state_status":      domain.DeploymentStatusFailed,
		"state_errcode":     reason.Error(),
		"state_started_at":  now,
		"state_finished_at": now,
	}).
		F("WHERE TRUE").
		S(
			builder.MaybeValue(criteria.App, "AND app_id = ?"),
			builder.MaybeValue(criteria.Target, "AND config_target = ?"),
			builder.MaybeValue(criteria.Status, "AND state_status = ?"),
			builder.MaybeValue(criteria.Environment, "AND config_environment = ?"),
		).
		Exec(s.db, ctx)
}

func (s *deploymentsStore) Write(c context.Context, deployments ...*domain.Deployment) error {
	return sqlite.WriteEvents(s.db, c, deployments,
		"[deployment.deployments]",
		func(d *domain.Deployment) sqlite.Key {
			return sqlite.Key{
				"app_id":            d.ID().AppID(),
				"deployment_number": d.ID().DeploymentNumber(),
			}
		},
		func(e event.Event, v builder.Values) sqlite.WriteMode {
			switch evt := e.(type) {
			case domain.DeploymentCreated:
				v["app_id"] = evt.ID.AppID()
				v["deployment_number"] = evt.ID.DeploymentNumber()
				v["config_appid"] = evt.Config.AppID()
				v["config_appname"] = evt.Config.AppName()
				v["config_environment"] = evt.Config.Environment()
				v["config_target"] = evt.Config.Target()
				v["config_vars"] = evt.Config.Vars()
				v["state_status"] = evt.State.Status()
				v["state_errcode"] = evt.State.ErrCode()
				v["state_services"] = evt.State.Services()
				v["state_started_at"] = evt.State.StartedAt()
				v["state_finished_at"] = evt.State.FinishedAt()
				v["source_discriminator"] = evt.Source.Kind()
				v["source"] = evt.Source
				v["requested_at"] = evt.Requested.At()
				v["requested_by"] = evt.Requested.By()
			case domain.DeploymentStateChanged:
				v["state_status"] = evt.State.Status()
				v["state_errcode"] = evt.State.ErrCode()
				v["state_services"] = evt.State.Services()
				v["state_started_at"] = evt.State.StartedAt()
				v["state_finished_at"] = evt.State.FinishedAt()
			}

			return sqlite.WriteModeUpsert
		})
}

type deploymentsOnAppTargetEnv struct {
	runningOrPending bool
	successful       bool
}

func deploymentsOnAppTargetEnvMapper(scanner storage.Scanner) (d deploymentsOnAppTargetEnv, err error) {
	err = scanner.Scan(&d.runningOrPending, &d.successful)

	return d, err
}
