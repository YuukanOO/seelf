package trigger

import (
	"context"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
)

type (
	Trigger interface {
		domain.Trigger
		CanPrepare(any) bool
		CanFetch(domain.Meta) bool
	}

	facade struct {
		triggers []Trigger
	}
)

// Creates a new facade which will call the appropriate trigger when calling Fetch or Prepare.
func NewFacade(triggers ...Trigger) domain.Trigger {
	return &facade{triggers}
}

func (r *facade) Prepare(app domain.App, payload any) (domain.Meta, error) {
	for _, trigger := range r.triggers {
		if trigger.CanPrepare(payload) {
			return trigger.Prepare(app, payload)
		}
	}

	return domain.Meta{}, domain.ErrNoValidTriggerFound
}

func (r *facade) Fetch(ctx context.Context, depl domain.Deployment) error {
	meta := depl.Trigger()

	for _, trigger := range r.triggers {
		if trigger.CanFetch(meta) {
			return trigger.Fetch(ctx, depl)
		}
	}

	return domain.ErrNoValidTriggerFound
}
