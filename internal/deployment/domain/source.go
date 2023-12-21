package domain

import (
	"context"
	"errors"

	"github.com/YuukanOO/seelf/pkg/storage"
)

var (
	ErrNoValidSourceFound   = errors.New("no_valid_source_found")
	ErrInvalidSourcePayload = errors.New("invalid_source_payload")

	SourceDataTypes = storage.NewDiscriminatedMapper(func(sd SourceData) string { return sd.Kind() })
)

type (
	// Contains stuff related to how the deployment has been triggered.
	// The inner data depends on the Source which has been requested.
	SourceData interface {
		Kind() string
		NeedVCS() bool
	}

	// Represents a source which has initiated a deployment.
	Source interface {
		Prepare(App, any) (SourceData, error)                              // Prepare the given payload for the given application, doing any needed validation
		Fetch(context.Context, string, DeploymentLogger, Deployment) error // Retrieve deployment data and store them in the given path before passing in to a backend
	}
)
