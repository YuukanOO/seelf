package sqlite

import (
	"context"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/event"
	"github.com/YuukanOO/seelf/pkg/storage/sqlite"
	"github.com/YuukanOO/seelf/pkg/storage/sqlite/builder"
)

type (
	DeploymentsStore interface {
		domain.DeploymentsReader
		domain.DeploymentsWriter
	}

	deploymentsStore struct {
		sqlite.Database
	}
)

func NewDeploymentsStore(db sqlite.Database) DeploymentsStore {
	return &deploymentsStore{db}
}

func (s *deploymentsStore) GetByID(ctx context.Context, id domain.DeploymentID) (domain.Deployment, error) {
	return builder.
		Query[domain.Deployment](`
			SELECT
				app_id
				,deployment_number
				,path
				,config_appname
				,config_environment
				,config_env
				,state_status
				,state_logfile
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
		One(s, ctx, domain.DeploymentFrom)
}

func (s *deploymentsStore) GetNextDeploymentNumber(ctx context.Context, appID domain.AppID) (domain.DeploymentNumber, error) {
	// FIXME: find a better way, on postgresql, I could have used a seq to increment the sequence to avoid any potential duplication
	// of a job number but on sqlite, I could not find a way yet.
	c, err := builder.
		Query[uint]("SELECT COUNT(*) FROM deployments WHERE app_id = ?", appID).
		Extract(s, ctx)

	if err != nil {
		return 0, err
	}

	return domain.DeploymentNumber(c + 1), nil
}

func (s *deploymentsStore) GetRunningDeployments(ctx context.Context) ([]domain.Deployment, error) {
	return builder.
		Query[domain.Deployment](`
		SELECT
			app_id
			,deployment_number
			,path
			,config_appname
			,config_environment
			,config_env
			,state_status
			,state_logfile
			,state_errcode
			,state_services
			,state_started_at
			,state_finished_at
			,source_discriminator
			,source
			,requested_at
			,requested_by
		FROM deployments
		WHERE state_status = ?`, domain.DeploymentStatusRunning).
		All(s, ctx, domain.DeploymentFrom)
}

func (s *deploymentsStore) GetRunningOrPendingDeploymentsCount(ctx context.Context, appID domain.AppID) (domain.RunningOrPendingAppDeploymentsCount, error) {
	return builder.
		Query[domain.RunningOrPendingAppDeploymentsCount](`
		SELECT COUNT(*)
		FROM deployments
		WHERE app_id = ? AND state_status IN (?, ?)`,
		appID, domain.DeploymentStatusRunning, domain.DeploymentStatusPending).
		Extract(s, ctx)
}

func (s *deploymentsStore) Write(c context.Context, deployments ...*domain.Deployment) error {
	return sqlite.WriteAndDispatch(s, c, deployments, func(ctx context.Context, e event.Event) error {
		switch evt := e.(type) {
		case domain.DeploymentCreated:
			return builder.
				Insert("deployments", builder.Values{
					"app_id":               evt.ID.AppID(),
					"deployment_number":    evt.ID.DeploymentNumber(),
					"path":                 evt.Path,
					"config_appname":       evt.Config.AppName(),
					"config_environment":   evt.Config.Environment(),
					"config_env":           evt.Config.Env(),
					"state_status":         evt.State.Status(),
					"state_logfile":        evt.State.LogFile(),
					"state_errcode":        evt.State.ErrCode(),
					"state_services":       evt.State.Services(),
					"state_started_at":     evt.State.StartedAt(),
					"state_finished_at":    evt.State.FinishedAt(),
					"source_discriminator": evt.Source.Discriminator(),
					"source":               evt.Source,
					"requested_at":         evt.Requested.At(),
					"requested_by":         evt.Requested.By(),
				}).
				Exec(s, ctx)
		case domain.DeploymentStateChanged:
			return builder.
				Update("deployments", builder.Values{
					"state_status":      evt.State.Status(),
					"state_logfile":     evt.State.LogFile(),
					"state_errcode":     evt.State.ErrCode(),
					"state_services":    evt.State.Services(),
					"state_started_at":  evt.State.StartedAt(),
					"state_finished_at": evt.State.FinishedAt(),
				}).
				F("WHERE app_id = ? AND deployment_number = ?", evt.ID.AppID(), evt.ID.DeploymentNumber()).
				Exec(s, ctx)
		default:
			return nil
		}
	})
}
