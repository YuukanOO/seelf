package memory

import (
	"context"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/event"
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
			availability = availability | domain.AppNamingTakenInProduction
		}

		if app.stagingTarget == staging {
			availability = availability | domain.AppNamingTakenInStaging
		}
	}

	if availability != 0 {
		return availability, nil
	}

	return domain.AppNamingAvailable, nil
}

func (s *appsStore) GetTargetAppNamingAvailability(
	ctx context.Context,
	id domain.AppID,
	env domain.Environment,
	target domain.TargetID,
) (domain.TargetAppNamingAvailability, error) {
	// Retrieve app name
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

	// And check if an app on the target and env already exists
	for _, app := range s.apps {
		if app.id == id || app.name != name {
			continue
		}

		switch env {
		case domain.Production:
			if app.productionTarget == target {
				return domain.TargetAppNamingTaken, nil
			}
		case domain.Staging:
			if app.stagingTarget == target {
				return domain.TargetAppNamingTaken, nil
			}
		}
	}

	return domain.TargetAppNamingAvailable, nil
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
