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

func (f *facade) Prepare(ctx context.Context, payload any, existing ...domain.ProviderConfig) (domain.ProviderConfig, error) {
	for _, p := range f.providers {
		if p.CanPrepare(payload) {
			return p.Prepare(ctx, payload, existing...)
		}
	}

	return nil, domain.ErrNoValidProviderFound
}

func (f *facade) Run(context.Context, domain.DeploymentContext, domain.Deployment) (domain.Services, error) {
	return nil, nil
}

func (f *facade) Stale(ctx context.Context, id domain.TargetID) error {
	for _, p := range f.providers {
		if err := p.Stale(ctx, id); err != nil {
			return err
		}
	}

	return nil
}

func (f *facade) CleanupTarget(ctx context.Context, target domain.Target) error {
	config := target.Provider()

	for _, p := range f.providers {
		if p.CanHandle(config) {
			return p.CleanupTarget(ctx, target)
		}
	}

	return nil
}

func (f *facade) Cleanup(context.Context, domain.App) error {
	return nil
}
