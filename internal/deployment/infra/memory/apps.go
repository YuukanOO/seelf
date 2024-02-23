package memory

import (
	"context"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/event"
	"github.com/YuukanOO/seelf/pkg/flag"
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

func (s *appsStore) GetAppNamingAvailability(
	ctx context.Context,
	name domain.AppName,
	production domain.TargetID,
	staging domain.TargetID,
) (domain.AppNamingAvailability, error) {
	var availability domain.AppNamingAvailability

	for _, app := range s.apps {
		if app.name != name {
			continue
		}

		if app.productionTarget == production {
			availability |= domain.AppNamingTakenInProduction
		}

		if app.stagingTarget == staging {
			availability |= domain.AppNamingTakenInStaging
		}
	}

	if !flag.IsSet(availability, domain.AppNamingTakenInProduction) {
		availability |= domain.AppNamingProductionAvailable
	}

	if !flag.IsSet(availability, domain.AppNamingTakenInStaging) {
		availability |= domain.AppNamingStagingAvailable
	}

	return availability, nil
}

func (s *appsStore) GetAppNamingAvailabilityOnID(
	ctx context.Context,
	id domain.AppID,
	production monad.Maybe[domain.TargetID],
	staging monad.Maybe[domain.TargetID],
) (domain.AppNamingAvailability, error) {
	productionTarget, hasProductionTarget := production.TryGet()
	stagingTarget, hasStagingTarget := staging.TryGet()

	// No input, no check!
	if !hasProductionTarget && !hasStagingTarget {
		return 0, nil
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
		return 0, apperr.ErrNotFound
	}

	var availability domain.AppNamingAvailability

	// And check if an app on the target and env already exists
	for _, app := range s.apps {
		if app.id == id || app.name != name {
			continue
		}

		if hasProductionTarget && app.productionTarget == productionTarget {
			availability |= domain.AppNamingTakenInProduction
		}

		if hasStagingTarget && app.stagingTarget == stagingTarget {
			availability |= domain.AppNamingTakenInStaging
		}
	}

	if !flag.IsSet(availability, domain.AppNamingTakenInProduction) {
		availability |= domain.AppNamingProductionAvailable
	}

	if !flag.IsSet(availability, domain.AppNamingTakenInStaging) {
		availability |= domain.AppNamingStagingAvailable
	}

	return availability, nil
}

func (s *appsStore) GetAppsOnTargetCount(ctx context.Context, target domain.TargetID) (domain.AppsOnTargetCount, error) {
	var count domain.AppsOnTargetCount

	for _, app := range s.apps {
		if app.productionTarget == target || app.stagingTarget == target {
			count++
		}
	}

	return count, nil
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
