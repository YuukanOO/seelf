package provider

import (
	"context"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
)

type (
	Provider interface {
		domain.Provider
		CanPrepare(any) bool
		CanHandle(domain.ProviderConfig) bool
	}

	facade struct {
		providers []Provider
	}
)

// Creates a new facade which will call the appropriate provider.
func NewFacade(providers ...Provider) domain.Provider {
	return &facade{providers}
}

func (f *facade) Prepare(ctx context.Context, payload any) (domain.ProviderConfig, error) {
	for _, p := range f.providers {
		if p.CanPrepare(payload) {
			return p.Prepare(ctx, payload)
		}
	}

	return nil, domain.ErrNoValidProviderFound
}

func (f *facade) Run(context.Context, domain.DeploymentContext, domain.Deployment) (domain.Services, error) {
	return nil, nil
}

func (f *facade) Cleanup(context.Context, domain.App) error {
	return nil
}
