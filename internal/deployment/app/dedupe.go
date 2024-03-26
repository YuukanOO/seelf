package app

import (
	"github.com/YuukanOO/seelf/internal/deployment/domain"
)

// Dedupe name for deployment to prevent multiple deployment at the same time on the same
// environment.
func DeploymentDedupeName(config domain.DeploymentConfig) string {
	return "deployment.deployment.deploy." + config.ProjectName()
}

// Dedupe name for app cleanup related tasks.
func AppCleanupDedupeName(appID domain.AppID) string {
	return "deployment.app.cleanup." + string(appID)
}

// Dedupe name for target operation to prevent multiple target configuration at the same time.
func TargetOperationDedupeName(id domain.TargetID) string {
	return "deployment.target.operation." + string(id)
}
