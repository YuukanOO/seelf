package app

import (
	"github.com/YuukanOO/seelf/internal/deployment/domain"
)

// Group for deployment to prevent multiple deployment at the same time on the same
// environment.
func DeploymentGroup(config domain.DeploymentConfig) string {
	return "deployment.deployment.deploy." + config.ProjectName()
}

// Group for target operation to prevent multiple target configuration at the same time.
func TargetConfigurationGroup(id domain.TargetID) string {
	return "deployment.target.configure." + string(id)
}
