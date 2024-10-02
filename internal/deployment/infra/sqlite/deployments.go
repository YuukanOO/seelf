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
		FROM deployments
		WHERE app_id = ? AND deployment_number = ?`, id.AppID(), id.DeploymentNumber()).
		One(s.db, ctx, domain.DeploymentFrom)
}

func (s *deploymentsStore) GetLastDeployment(ctx context.Context, id domain.AppID, env domain.Environment) (domain.Deployment, error) {
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
		FROM deployments
		WHERE app_id = ? AND config_environment = ?
		ORDER BY deployment_number DESC
		LIMIT 1`, id, env).
		One(s.db, ctx, domain.DeploymentFrom)
}

func (s *deploymentsStore) GetNextDeploymentNumber(ctx context.Context, appID domain.AppID) (domain.DeploymentNumber, error) {
	// FIXME: find a better way, on postgresql, I could have used a seq to increment the sequence to avoid any potential duplication
	// of a job number but on sqlite, I could not find a way yet.
	c, err := builder.
		Query[uint]("SELECT COUNT(*) FROM deployments WHERE app_id = ?", appID).
		Extract(s.db, ctx)

	if err != nil {
		return 0, err
	}

	return domain.DeploymentNumber(c + 1), nil
}

func (s *deploymentsStore) HasRunningOrPendingDeploymentsOnTarget(ctx context.Context, target domain.TargetID) (domain.HasRunningOrPendingDeploymentsOnTarget, error) {
	r, err := builder.
		Query[bool](`
		SELECT EXISTS(SELECT 1 FROM deployments WHERE config_target = ? AND state_status IN (?, ?))`,
		target, domain.DeploymentStatusRunning, domain.DeploymentStatusPending).
		Extract(s.db, ctx)

	return domain.HasRunningOrPendingDeploymentsOnTarget(r), err
}

func (s *deploymentsStore) HasDeploymentsOnAppTargetEnv(ctx context.Context, app domain.AppID, target domain.TargetID, env domain.Environment, ti shared.TimeInterval) (
	domain.HasRunningOrPendingDeploymentsOnAppTargetEnv,
	domain.HasSuccessfulDeploymentsOnAppTargetEnv,
	error,
) {
	c, err := builder.
		Query[deploymentsOnAppTargetEnv](`
		SELECT
			EXISTS(
				SELECT 1 FROM deployments
				WHERE 
					app_id = ? AND config_target = ? AND config_environment = ?
					AND state_status IN (?, ?)
			) AS runningOrPending
			,EXISTS(
				SELECT 1 FROM deployments
				WHERE
					app_id = ? AND config_target = ? AND config_environment = ?
					AND state_status = ? AND requested_at >= ? AND requested_at <= ?
			) AS successful`,
		app, target, env, domain.DeploymentStatusPending, domain.DeploymentStatusRunning,
		app, target, env, domain.DeploymentStatusSucceeded, ti.From(), ti.To()).
		One(s.db, ctx, deploymentsOnAppTargetEnvMapper)

	return domain.HasRunningOrPendingDeploymentsOnAppTargetEnv(c.runningOrPending),
		domain.HasSuccessfulDeploymentsOnAppTargetEnv(c.successful), err
}

func (s *deploymentsStore) FailDeployments(ctx context.Context, reason error, criteria domain.FailCriteria) error {
	now := time.Now().UTC()

	return builder.Update("deployments", builder.Values{
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
	return sqlite.WriteEvents(s.db, c, deployments, func(ctx context.Context, e event.Event) error {
		switch evt := e.(type) {
		case domain.DeploymentCreated:
			return builder.
				Insert("deployments", builder.Values{
					"app_id":               evt.ID.AppID(),
					"deployment_number":    evt.ID.DeploymentNumber(),
					"config_appid":         evt.Config.AppID(),
					"config_appname":       evt.Config.AppName(),
					"config_environment":   evt.Config.Environment(),
					"config_target":        evt.Config.Target(),
					"config_vars":          evt.Config.Vars(),
					"state_status":         evt.State.Status(),
					"state_errcode":        evt.State.ErrCode(),
					"state_services":       evt.State.Services(),
					"state_started_at":     evt.State.StartedAt(),
					"state_finished_at":    evt.State.FinishedAt(),
					"source_discriminator": evt.Source.Kind(),
					"source":               evt.Source,
					"requested_at":         evt.Requested.At(),
					"requested_by":         evt.Requested.By(),
				}).
				Exec(s.db, ctx)
		case domain.DeploymentStateChanged:
			return builder.
				Update("deployments", builder.Values{
					"state_status":      evt.State.Status(),
					"state_errcode":     evt.State.ErrCode(),
					"state_services":    evt.State.Services(),
					"state_started_at":  evt.State.StartedAt(),
					"state_finished_at": evt.State.FinishedAt(),
				}).
				F("WHERE app_id = ? AND deployment_number = ?", evt.ID.AppID(), evt.ID.DeploymentNumber()).
				Exec(s.db, ctx)
		default:
			return nil
		}
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
