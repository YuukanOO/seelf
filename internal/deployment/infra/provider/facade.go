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

func (f *facade) Run(ctx context.Context, info domain.DeploymentContext, depl domain.Deployment, target domain.Target) (domain.Services, error) {
	provider, err := f.providerForTarget(target)

	if err != nil {
		return nil, err
	}

	return provider.Run(ctx, info, depl, target)
}

func (f *facade) Configure(ctx context.Context, target domain.Target) error {
	provider, err := f.providerForTarget(target)

	if err != nil {
		return err
	}

	return provider.Configure(ctx, target)
}

func (f *facade) CleanupTarget(ctx context.Context, target domain.Target, strategy domain.TargetCleanupStrategy) error {
	provider, err := f.providerForTarget(target)

	if err != nil {
		return err
	}

	return provider.CleanupTarget(ctx, target, strategy)
}

func (f *facade) Cleanup(ctx context.Context, app domain.AppID, target domain.Target, env domain.Environment) error {
	provider, err := f.providerForTarget(target)

	if err != nil {
		return err
	}

	return provider.Cleanup(ctx, app, target, env)
}

func (f *facade) providerForTarget(target domain.Target) (Provider, error) {
	config := target.Provider()

	for _, p := range f.providers {
		if p.CanHandle(config) {
			return p, nil
		}
	}

	return nil, domain.ErrNoValidProviderFound
}
