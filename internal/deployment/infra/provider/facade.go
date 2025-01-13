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

func (f *facade) Deploy(ctx context.Context, info domain.DeploymentContext, deployment domain.Deployment, target domain.Target, registries []domain.Registry) (domain.Services, error) {
	provider, err := f.providerForTarget(target)

	if err != nil {
		return nil, err
	}

	return provider.Deploy(ctx, info, deployment, target, registries)
}

func (f *facade) Setup(ctx context.Context, target domain.Target) (domain.TargetEntrypointsAssigned, error) {
	provider, err := f.providerForTarget(target)

	if err != nil {
		return nil, err
	}

	return provider.Setup(ctx, target)
}

func (f *facade) RemoveConfiguration(ctx context.Context, target domain.TargetID) error {
	for _, p := range f.providers {
		if err := p.RemoveConfiguration(ctx, target); err != nil {
			return err
		}
	}

	return nil
}

func (f *facade) CleanupTarget(ctx context.Context, target domain.Target, strategy domain.CleanupStrategy) error {
	provider, err := f.providerForTarget(target)

	if err != nil {
		return err
	}

	return provider.CleanupTarget(ctx, target, strategy)
}

func (f *facade) Cleanup(ctx context.Context, app domain.AppID, target domain.Target, env domain.EnvironmentName, strategy domain.CleanupStrategy) error {
	provider, err := f.providerForTarget(target)

	if err != nil {
		return err
	}

	return provider.Cleanup(ctx, app, target, env, strategy)
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
