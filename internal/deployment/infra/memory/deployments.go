package memory

import (
	"context"
	"slices"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/event"
	"golang.org/x/exp/maps"
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

func (s *deploymentsStore) GetNextDeploymentNumber(ctx context.Context, appid domain.AppID) (domain.DeploymentNumber, error) {
	count := 0

	for _, depl := range s.deployments {
		if depl.id.AppID() == appid {
			count += 1
		}
	}

	return domain.DeploymentNumber(count + 1), nil
}

func (s *deploymentsStore) GetLatestSuccessfulDeployments(ctx context.Context, appid domain.AppID) ([]domain.Deployment, error) {
	group := make(map[domain.Environment]domain.Deployment)

	for _, depl := range s.deployments {
		if depl.id.AppID() != appid || depl.state.Status() != domain.DeploymentStatusSucceeded {
			continue
		}

		env := depl.value.Config().Environment()

		if d, exists := group[env]; !exists || depl.id.DeploymentNumber() > d.ID().DeploymentNumber() {
			group[env] = *depl.value
		}
	}

	return maps.Values(group), nil
}

func (s *deploymentsStore) GetRunningDeploymentsOnTargetCount(ctx context.Context, id domain.TargetID) (domain.RunningDeploymentsOnTargetCount, error) {
	var count domain.RunningDeploymentsOnTargetCount

	for _, depl := range s.deployments {
		if depl.value.Config().Target() == id && depl.state.Status() == domain.DeploymentStatusRunning {
			count += 1
		}
	}

	return count, nil
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

func (s *deploymentsStore) FailDeployments(ctx context.Context, status domain.DeploymentStatus, reason error, appids ...domain.AppID) error {
	for _, d := range s.deployments {
		if (len(appids) > 0 && !slices.Contains(appids, d.id.AppID())) ||
			(d.state.Status() != status) {
			continue
		}

		d.value.HasStarted() // try the has started to make sure it is started
		d.value.HasEnded(domain.Services{}, reason)
	}

	return nil
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
