package domain

import "context"

type (
	// Backend service used to run an application services.
	Backend interface {
		Run(context.Context, Deployment) (Services, error) // Launch a deployment through the backend and return services that has been deployed
		Cleanup(context.Context, App) error                // Cleanup an application, which means removing every possible stuff related to it
	}

	// Specific logger interface use by deployment jobs to document the deployment process.
	StepLogger interface {
		Stepf(string, ...any)
		Warnf(string, ...any)
		Infof(string, ...any)
		Error(error)
	}
)
