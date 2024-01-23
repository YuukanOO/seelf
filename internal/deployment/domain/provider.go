package domain

import (
	"context"
)

type (
	// Provider specific configuration.
	ProviderConfig interface {
		Kind() string
		Fingerprint() string // Used for unicity of a provider configuration (such as one per host)
	}

	// Provider used to run an application services.
	Provider interface {
		Prepare(context.Context, any) (ProviderConfig, error)                 // Prepare the given payload representing a Provider specific configuration
		Run(context.Context, DeploymentContext, Deployment) (Services, error) // Launch a deployment and return services that has been deployed
		Cleanup(context.Context, App) error                                   // Cleanup an application, which means removing every possible stuff related to it
	}
)
