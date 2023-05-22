package memory

import (
	"context"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/collections"
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
		id    domain.AppID
		name  domain.UniqueAppName
		value *domain.App
	}
)

func NewAppsStore(existingApps ...domain.App) AppsStore {
	s := &appsStore{}
	ctx := context.Background()

	s.Write(ctx, collections.ToPointers(existingApps)...)

	return s
}

func (s *appsStore) IsNameUnique(ctx context.Context, name domain.AppName) (domain.UniqueAppName, error) {
	uniqueName := domain.UniqueAppName(name)
	for _, app := range s.apps {
		if app.name == uniqueName {
			return "", domain.ErrAppNameAlreadyTaken
		}
	}

	return uniqueName, nil
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
					id:    evt.ID,
					name:  evt.Name,
					value: app,
				})
			case domain.AppDeleted:
				for i, a := range s.apps {
					if a.id == app.ID() {
						s.apps = append(s.apps[:i], s.apps[i+1:]...)
						break
					}
				}
			default:
				for _, a := range s.apps {
					if a.id == app.ID() {
						a.value = app
						break
					}
				}
			}
		}
	}

	return nil
}
