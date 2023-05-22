package domain

import "context"

// Backend service used to run an application services.
type Backend interface {
	Run(context.Context, Deployment) (Services, error) // Launch a deployment through the backend and return services that has been deployed
	Cleanup(context.Context, App) error                // Cleanup an application, which means removing every possible stuff related to it
}
