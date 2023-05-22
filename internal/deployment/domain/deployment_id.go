package domain

// The deployment unique identifier is a composite key
// based on the app id and the deployment number.
type (
	DeploymentNumber int

	DeploymentID struct {
		appID            AppID
		deploymentNumber DeploymentNumber
	}
)

// Construct a deployment id from an app and a deployment number
func DeploymentIDFrom(app AppID, number DeploymentNumber) DeploymentID {
	return DeploymentID{app, number}
}

func (i DeploymentID) AppID() AppID                       { return i.appID }
func (i DeploymentID) DeploymentNumber() DeploymentNumber { return i.deploymentNumber }
