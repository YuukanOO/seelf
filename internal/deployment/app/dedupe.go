package app

import (
	"fmt"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
)

// Dedupe name for deployment to prevent multiple deployment at the same time on the same
// environment.
func DeploymentDedupeName(config domain.DeploymentConfig) string {
	return fmt.Sprintf("deployment.deployment.deploy.%s", config.ProjectName())
}

// Dedupe name for app cleanup related tasks.
func AppCleanupDedupeName(appID domain.AppID) string {
	return fmt.Sprintf("deployment.app.cleanup.%s", appID)
}

// Dedupe name for target operation to prevent multiple target configuration at the same time.
func TargetOperationDedupeName(id domain.TargetID) string {
	return fmt.Sprintf("deployment.target.operation.%s", id)
}
