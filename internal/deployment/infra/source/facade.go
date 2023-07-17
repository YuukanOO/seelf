package source

import (
	"context"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
)

type (
	Source interface {
		domain.Source
		CanPrepare(any) bool
		CanFetch(domain.Meta) bool
	}

	facade struct {
		sources []Source
	}
)

// Creates a new facade which will call the appropriate source when calling Fetch or Prepare.
func NewFacade(sources ...Source) domain.Source {
	return &facade{sources}
}

func (r *facade) Prepare(app domain.App, payload any) (domain.Meta, error) {
	for _, src := range r.sources {
		if src.CanPrepare(payload) {
			return src.Prepare(app, payload)
		}
	}

	return domain.Meta{}, domain.ErrNoValidSourceFound
}

func (r *facade) Fetch(ctx context.Context, depl domain.Deployment) error {
	meta := depl.Source()

	for _, src := range r.sources {
		if src.CanFetch(meta) {
			return src.Fetch(ctx, depl)
		}
	}

	return domain.ErrNoValidSourceFound
}
