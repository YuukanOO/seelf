package memory

import (
	"context"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/collections"
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
		state domain.State
	}
)

func NewDeploymentsStore(existingDeployments ...domain.Deployment) DeploymentsStore {
	s := &deploymentsStore{}
	ctx := context.Background()

	s.Write(ctx, collections.ToPointers(existingDeployments)...)

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

func (s *deploymentsStore) GetNextDeploymentNumber(ctx context.Context, appid domain.AppID) (domain.DeploymentNumber, error) {
	count := 0

	for _, depl := range s.deployments {
		if depl.id.AppID() == appid {
			count += 1
		}
	}

	return domain.DeploymentNumber(count + 1), nil
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
						d.value = depl
						d.state = evt.State
						break
					}
				}
			default:
				for _, d := range s.deployments {
					if d.id == depl.ID() {
						d.value = depl
						break
					}
				}
			}
		}
	}

	return nil
}

func (s *deploymentsStore) GetRunningDeployments(context.Context) ([]domain.Deployment, error) {
	var result []domain.Deployment

	for _, d := range s.deployments {
		if d.state.Status() == domain.DeploymentStatusRunning {
			result = append(result, *d.value)
		}
	}

	return result, nil
}

func (s *deploymentsStore) GetRunningOrPendingDeploymentsCount(ctx context.Context, appid domain.AppID) (domain.RunningOrPendingAppDeploymentsCount, error) {
	var count domain.RunningOrPendingAppDeploymentsCount

	for _, d := range s.deployments {
		if d.id.AppID() == appid && (d.state.Status() == domain.DeploymentStatusRunning || d.state.Status() == domain.DeploymentStatusPending) {
			count += 1
		}
	}

	return count, nil
}
