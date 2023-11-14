package domain

import "io"

// Specific logger interface use by deployment jobs to document the deployment process.
type DeploymentLogger interface {
	io.WriteCloser

	Stepf(string, ...any)
	Warnf(string, ...any)
	Infof(string, ...any)
	Error(error)
}
