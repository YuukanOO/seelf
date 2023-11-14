package domain

import (
	"context"
)

// Backend service used to run an application services.
type Backend interface {
	Run(context.Context, string, DeploymentLogger, Deployment) (Services, error) // Launch a deployment stored in the given path through the backend and return services that has been deployed
	Cleanup(context.Context, App) error                                          // Cleanup an application, which means removing every possible stuff related to it
}
