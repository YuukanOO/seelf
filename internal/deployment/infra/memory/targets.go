package memory

import (
	"context"
	"slices"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/event"
)

type (
	TargetsStore interface {
		domain.TargetsReader
		domain.TargetsWriter
	}

	targetsStore struct {
		targets []*targetData
	}

	targetData struct {
		id     domain.TargetID
		domain domain.Url
		value  *domain.Target
	}
)

func NewTargetsStore(existingTargets ...*domain.Target) TargetsStore {
	s := &targetsStore{}

	s.Write(context.Background(), existingTargets...)

	return s
}

func (s *targetsStore) GetDomainAvailability(ctx context.Context, domainUrl domain.Url, excluded ...domain.TargetID) (domain.TargetDomainAvailability, error) {
	var target *domain.Target

	for _, t := range s.targets {
		if t.domain.String() == domainUrl.String() {
			target = t.value
			break
		}
	}

	if target == nil || slices.Contains(excluded, target.ID()) {
		return true, nil
	}

	return false, nil
}

func (s *targetsStore) GetConfigAvailability(ctx context.Context, config domain.ProviderConfig, excluded ...domain.TargetID) (domain.TargetConfigAvailability, error) {
	var target *domain.Target

	for _, t := range s.targets {
		if t.value.Provider().Fingerprint() == config.Fingerprint() {
			target = t.value
			break
		}
	}

	if target == nil || slices.Contains(excluded, target.ID()) {
		return true, nil
	}

	return false, nil
}

func (s *targetsStore) GetByID(ctx context.Context, id domain.TargetID) (domain.Target, error) {
	for _, t := range s.targets {
		if t.id == id {
			return *t.value, nil
		}
	}

	return domain.Target{}, apperr.ErrNotFound
}

func (s *targetsStore) Write(ctx context.Context, targets ...*domain.Target) error {
	for _, target := range targets {
		for _, e := range event.Unwrap(target) {
			switch evt := e.(type) {
			case domain.TargetCreated:
				var exist bool
				for _, a := range s.targets {
					if a.id == evt.ID {
						exist = true
						break
					}
				}

				if exist {
					continue
				}

				s.targets = append(s.targets, &targetData{
					id:     evt.ID,
					domain: evt.Domain,
					value:  target,
				})
			case domain.TargetDomainChanged:
				for _, t := range s.targets {
					if t.id == evt.ID {
						t.domain = evt.Domain
						*t.value = *target
						break
					}
				}
			case domain.TargetDeleted:
				for i, t := range s.targets {
					if t.id == target.ID() {
						*t.value = *target
						s.targets = append(s.targets[:i], s.targets[i+1:]...)
						break
					}
				}
			default:
				for _, t := range s.targets {
					if t.id == target.ID() {
						*t.value = *target
						break
					}
				}
			}
		}
	}

	return nil
}
