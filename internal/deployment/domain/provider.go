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
		String() string             // User friendly representation, mostly for logs
	}

	// Provider used to run an application services.
	Provider interface {
		// Prepare the given payload representing a Provider specific configuration.
		Prepare(ctx context.Context, payload any, existing ...ProviderConfig) (ProviderConfig, error)
		// Run a deployment on the specified target and return services that has been deployed.
		Run(context.Context, DeploymentContext, Deployment, Target) (Services, error)
		// Configure a target by deploying the needed stuff to actually serve deployments.
		Configure(context.Context, Target) error
		// Cleanup a target, removing every resources managed by seelf on it.
		// The last parameter indicate if the cleanup is forced which means
		// it should always succeed and remove everything the provider can.
		CleanupTarget(context.Context, Target, TargetCleanupStrategy) error
		// Cleanup an application on the specified target and environment, which means removing every possible stuff related to it
		Cleanup(context.Context, AppID, Target, Environment) error
	}
)
