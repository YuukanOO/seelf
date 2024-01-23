package source

import (
	"context"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
)

type (
	Source interface {
		domain.Source
		CanPrepare(any) bool
		CanFetch(domain.SourceData) bool
	}

	facade struct {
		sources []Source
	}
)

// Creates a new facade which will call the appropriate source when calling Fetch or Prepare.
func NewFacade(sources ...Source) domain.Source {
	return &facade{sources}
}

func (r *facade) Prepare(ctx context.Context, app domain.App, payload any) (domain.SourceData, error) {
	for _, src := range r.sources {
		if src.CanPrepare(payload) {
			return src.Prepare(ctx, app, payload)
		}
	}

	return nil, domain.ErrNoValidSourceFound
}

func (r *facade) Fetch(ctx context.Context, deploymentCtx domain.DeploymentContext, depl domain.Deployment) error {
	meta := depl.Source()

	for _, src := range r.sources {
		if src.CanFetch(meta) {
			return src.Fetch(ctx, deploymentCtx, depl)
		}
	}

	return domain.ErrNoValidSourceFound
}
