package memory

import (
	"context"
	"slices"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/event"
)

type (
	RegistriesStore interface {
		domain.RegistriesReader
		domain.RegistriesWriter
	}

	registriesStore struct {
		registries []*registryData
	}

	registryData struct {
		id    domain.RegistryID
		value *domain.Registry
	}
)

func NewRegistriesStore(existingApps ...*domain.Registry) RegistriesStore {
	s := &registriesStore{}

	s.Write(context.Background(), existingApps...)

	return s
}

func (s *registriesStore) CheckUrlAvailability(ctx context.Context, domainUrl domain.Url, excluded ...domain.RegistryID) (domain.RegistryUrlRequirement, error) {
	var registry *domain.Registry

	for _, t := range s.registries {
		if t.value.Url() == domainUrl {
			registry = t.value
			break
		}
	}

	return domain.NewRegistryUrlRequirement(domainUrl, registry == nil || slices.Contains(excluded, registry.ID())), nil
}

func (s *registriesStore) GetByID(ctx context.Context, id domain.RegistryID) (domain.Registry, error) {
	for _, r := range s.registries {
		if r.id == id {
			return *r.value, nil
		}
	}

	return domain.Registry{}, apperr.ErrNotFound
}

func (s *registriesStore) GetAll(ctx context.Context) ([]domain.Registry, error) {
	var registries []domain.Registry

	for _, r := range s.registries {
		registries = append(registries, *r.value)
	}

	return registries, nil
}

func (s *registriesStore) Write(ctx context.Context, registries ...*domain.Registry) error {
	for _, reg := range registries {
		for _, e := range event.Unwrap(reg) {
			switch evt := e.(type) {
			case domain.RegistryCreated:
				var exist bool
				for _, r := range s.registries {
					if r.id == evt.ID {
						exist = true
						break
					}
				}

				if exist {
					continue
				}

				s.registries = append(s.registries, &registryData{
					id:    evt.ID,
					value: reg,
				})
			case domain.RegistryDeleted:
				for i, r := range s.registries {
					if r.id == reg.ID() {
						*r.value = *reg
						s.registries = append(s.registries[:i], s.registries[i+1:]...)
						break
					}
				}
			default:
				for _, r := range s.registries {
					if r.id == reg.ID() {
						*r.value = *reg
						break
					}
				}
			}
		}
	}

	return nil
}
