package domain

import (
	"context"
)

// Manage all build artifacts.
type ArtifactManager interface {
	// Prepare the build directory and logger for the given deployment.
	// Returns the build directory path and the logger to use for each of the deployment steps.
	// You MUST close the logger if no err has been returned.
	PrepareBuild(context.Context, Deployment) (string, DeploymentLogger, error)
	// Cleanup an application artefacts.
	Cleanup(context.Context, App) error
	// Returns the absolute path to a deployment log file.
	LogPath(context.Context, Deployment) string
}
