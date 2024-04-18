package domain

import (
	"context"
)

type (
	// Specific context for a deployment.
	DeploymentContext struct {
		directory string
		logger    DeploymentLogger
	}

	// Manage all build artifacts.
	ArtifactManager interface {
		// Prepare the build directory and logger for the given deployment.
		// You MUST close the Logger if no err has been returned.
		PrepareBuild(context.Context, Deployment) (DeploymentContext, error)
		// Cleanup an application artifacts.
		Cleanup(context.Context, AppID) error
		// Returns the absolute path to a deployment log file.
		LogPath(context.Context, Deployment) string
	}
)

// Builds up a new DeploymentContext used by deployment participants.
func NewDeploymentContext(buildDirectory string, logger DeploymentLogger) DeploymentContext {
	return DeploymentContext{
		directory: buildDirectory,
		logger:    logger,
	}
}

func (d DeploymentContext) BuildDirectory() string   { return d.directory }
func (d DeploymentContext) Logger() DeploymentLogger { return d.logger }
