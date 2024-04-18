package memory

import (
	"context"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/apperr"
	shared "github.com/YuukanOO/seelf/pkg/domain"
	"github.com/YuukanOO/seelf/pkg/event"
)

type (
	DeploymentsStore interface {
		domain.DeploymentsReader
		domain.DeploymentsWriter
	}

	deploymentsStore struct {
		deployments []*deploymentData
	}

	deploymentData struct {
		id    domain.DeploymentID
		value *domain.Deployment
		state domain.DeploymentState
	}
)

func NewDeploymentsStore(existingDeployments ...*domain.Deployment) DeploymentsStore {
	s := &deploymentsStore{}

	s.Write(context.Background(), existingDeployments...)

	return s
}

func (s *deploymentsStore) GetByID(ctx context.Context, id domain.DeploymentID) (domain.Deployment, error) {
	for _, depl := range s.deployments {
		if depl.id == id {
			return *depl.value, nil
		}
	}

	return domain.Deployment{}, apperr.ErrNotFound
}

func (s *deploymentsStore) GetLastDeployment(ctx context.Context, id domain.AppID, env domain.Environment) (domain.Deployment, error) {
	var last *deploymentData

	for _, depl := range s.deployments {
		if depl.id.AppID() == id && depl.value.Config().Environment() == env {
			if last == nil || last.id.DeploymentNumber() < depl.id.DeploymentNumber() {
				last = depl
			}
		}
	}

	if last == nil {
		return domain.Deployment{}, apperr.ErrNotFound
	}

	return *last.value, nil

}

func (s *deploymentsStore) GetNextDeploymentNumber(ctx context.Context, appid domain.AppID) (domain.DeploymentNumber, error) {
	count := 0

	for _, depl := range s.deployments {
		if depl.id.AppID() == appid {
			count += 1
		}
	}

	return domain.DeploymentNumber(count + 1), nil
}

func (s *deploymentsStore) HasRunningOrPendingDeploymentsOnTarget(ctx context.Context, target domain.TargetID) (domain.HasRunningOrPendingDeploymentsOnTarget, error) {
	for _, d := range s.deployments {
		if d.value.Config().Target() == target && (d.state.Status() == domain.DeploymentStatusRunning || d.state.Status() == domain.DeploymentStatusPending) {
			return true, nil
		}
	}

	return false, nil
}

func (s *deploymentsStore) HasDeploymentsOnAppTargetEnv(ctx context.Context, app domain.AppID, target domain.TargetID, env domain.Environment, ti shared.TimeInterval) (
	domain.HasRunningOrPendingDeploymentsOnAppTargetEnv,
	domain.HasSuccessfulDeploymentsOnAppTargetEnv,
	error,
) {
	var (
		ongoing    domain.HasRunningOrPendingDeploymentsOnAppTargetEnv
		successful domain.HasSuccessfulDeploymentsOnAppTargetEnv
	)

	for _, d := range s.deployments {
		if d.id.AppID() != app || d.value.Config().Target() != target || d.value.Config().Environment() != env {
			continue
		}

		switch d.state.Status() {
		case domain.DeploymentStatusSucceeded:
			if d.value.Requested().At().After(ti.From()) && d.value.Requested().At().Before(ti.To()) {
				successful = true
			}
		case domain.DeploymentStatusRunning, domain.DeploymentStatusPending:
			ongoing = true
		}
	}

	return ongoing, successful, nil
}

func (s *deploymentsStore) FailDeployments(ctx context.Context, reason error, criterias domain.FailCriterias) error {
	panic("not implemented")
}

func (s *deploymentsStore) Write(ctx context.Context, deployments ...*domain.Deployment) error {
	for _, depl := range deployments {
		for _, e := range event.Unwrap(depl) {
			switch evt := e.(type) {
			case domain.DeploymentCreated:
				var exist bool
				for _, a := range s.deployments {
					if a.id == evt.ID {
						exist = true
						break
					}
				}

				if exist {
					continue
				}

				s.deployments = append(s.deployments, &deploymentData{
					id:    evt.ID,
					value: depl,
					state: evt.State,
				})
			case domain.DeploymentStateChanged:
				for _, d := range s.deployments {
					if d.id == depl.ID() {
						*d.value = *depl
						d.state = evt.State
						break
					}
				}
			default:
				for _, d := range s.deployments {
					if d.id == depl.ID() {
						*d.value = *depl
						break
					}
				}
			}
		}
	}

	return nil
}
