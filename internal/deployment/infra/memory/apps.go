package memory

import (
	"context"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/event"
	"github.com/YuukanOO/seelf/pkg/monad"
)

type (
	AppsStore interface {
		domain.AppsReader
		domain.AppsWriter
	}

	appsStore struct {
		apps []*appData
	}

	appData struct {
		id               domain.AppID
		name             domain.AppName
		productionTarget domain.TargetID
		stagingTarget    domain.TargetID
		value            *domain.App
	}
)

func NewAppsStore(existingApps ...*domain.App) AppsStore {
	s := &appsStore{}

	s.Write(context.Background(), existingApps...)

	return s
}

func (s *appsStore) CheckAppNamingAvailability(
	ctx context.Context,
	name domain.AppName,
	production domain.EnvironmentConfig,
	staging domain.EnvironmentConfig,
) (domain.EnvironmentConfigRequirement, domain.EnvironmentConfigRequirement, error) {
	var productionTaken, stagingTaken bool

	for _, app := range s.apps {
		if app.name != name {
			continue
		}

		if app.productionTarget == production.Target() {
			productionTaken = true
		}

		if app.stagingTarget == staging.Target() {
			stagingTaken = true
		}
	}

	return domain.NewEnvironmentConfigRequirement(production, true, !productionTaken),
		domain.NewEnvironmentConfigRequirement(staging, true, !stagingTaken),
		nil
}

func (s *appsStore) CheckAppNamingAvailabilityByID(
	ctx context.Context,
	id domain.AppID,
	production monad.Maybe[domain.EnvironmentConfig],
	staging monad.Maybe[domain.EnvironmentConfig],
) (
	productionRequirement domain.EnvironmentConfigRequirement,
	stagingRequirement domain.EnvironmentConfigRequirement,
	err error,
) {
	productionValue, hasProductionTarget := production.TryGet()
	stagingValue, hasStagingTarget := staging.TryGet()

	// No input, no check!
	if !hasProductionTarget && !hasStagingTarget {
		return productionRequirement, stagingRequirement, nil
	}

	// Retrieve app name by its ID
	var name domain.AppName

	for _, app := range s.apps {
		if app.id == id {
			name = app.name
			break
		}
	}

	if name == "" {
		return productionRequirement, stagingRequirement, apperr.ErrNotFound
	}

	var productionTaken, stagingTaken bool

	// And check if an app on the target and env already exists
	for _, app := range s.apps {
		if app.id == id || app.name != name {
			continue
		}

		if hasProductionTarget && app.productionTarget == productionValue.Target() {
			productionTaken = true
		}

		if hasStagingTarget && app.stagingTarget == stagingValue.Target() {
			stagingTaken = true
		}
	}

	if hasProductionTarget {
		productionRequirement = domain.NewEnvironmentConfigRequirement(productionValue, true, !productionTaken)
	}

	if hasStagingTarget {
		stagingRequirement = domain.NewEnvironmentConfigRequirement(stagingValue, true, !stagingTaken)
	}

	return productionRequirement, stagingRequirement, nil
}

func (s *appsStore) HasAppsOnTarget(ctx context.Context, target domain.TargetID) (domain.HasAppsOnTarget, error) {
	for _, app := range s.apps {
		if app.productionTarget == target || app.stagingTarget == target {
			return true, nil
		}
	}

	return false, nil
}

func (s *appsStore) GetByID(ctx context.Context, id domain.AppID) (domain.App, error) {
	for _, app := range s.apps {
		if app.id == id {
			return *app.value, nil
		}
	}

	return domain.App{}, apperr.ErrNotFound
}

func (s *appsStore) Write(ctx context.Context, apps ...*domain.App) error {
	for _, app := range apps {
		for _, e := range event.Unwrap(app) {
			switch evt := e.(type) {
			case domain.AppCreated:
				var exist bool
				for _, a := range s.apps {
					if a.id == evt.ID {
						exist = true
						break
					}
				}

				if exist {
					continue
				}

				s.apps = append(s.apps, &appData{
					id:               evt.ID,
					name:             evt.Name,
					productionTarget: evt.Production.Target(),
					stagingTarget:    evt.Staging.Target(),
					value:            app,
				})
			case domain.AppEnvChanged:
				for _, a := range s.apps {
					if a.id == app.ID() {
						switch evt.Environment {
						case domain.Production:
							a.productionTarget = evt.Config.Target()
						case domain.Staging:
							a.stagingTarget = evt.Config.Target()
						}
						*a.value = *app
						break
					}
				}
			case domain.AppDeleted:
				for i, a := range s.apps {
					if a.id == app.ID() {
						*a.value = *app
						s.apps = append(s.apps[:i], s.apps[i+1:]...)
						break
					}
				}
			default:
				for _, a := range s.apps {
					if a.id == app.ID() {
						*a.value = *app
						break
					}
				}
			}
		}
	}

	return nil
}
