package domain

import (
	"context"

	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/storage"
)

var (
	ErrNoValidProviderFound   = apperr.New("no_valid_provider_found")
	ErrInvalidProviderPayload = apperr.New("invalid_provider_payload")

	ProviderConfigTypes = storage.NewDiscriminatedMapper(func(c ProviderConfig) string { return c.Kind() })
)

type (
	// Provider specific configuration.
	ProviderConfig interface {
		Kind() string
		Equals(ProviderConfig) bool // Compare the provider configuration with another one
		Fingerprint() string        // Used for unicity of a provider configuration (such as one per host)
	}

	// Provider used to run an application services.
	Provider interface {
		Prepare(ctx context.Context, payload any, existing ...ProviderConfig) (ProviderConfig, error) // Prepare the given payload representing a Provider specific configuration
		Run(context.Context, DeploymentContext, Deployment) (Services, error)                         // Launch a deployment and return services that has been deployed
		Stale(context.Context, TargetID) error                                                        // Mark a target as stale, meaning it should be reinitialized before being used again
		CleanupTarget(context.Context, Target) error                                                  // Cleanup a target, removing every resources managed by seelf
		Cleanup(context.Context, App) error                                                           // Cleanup an application, which means removing every possible stuff related to it
	}
)
