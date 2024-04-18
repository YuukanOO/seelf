package domain

import (
	"context"

	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/storage"
)

var (
	ErrNoValidSourceFound   = apperr.New("no_valid_source_found")
	ErrInvalidSourcePayload = apperr.New("invalid_source_payload")

	SourceDataTypes = storage.NewDiscriminatedMapper(func(sd SourceData) string { return sd.Kind() })
)

type (
	// Contains stuff related to how the deployment has been triggered.
	// The inner data depends on the Source which has been requested.
	SourceData interface {
		Kind() string
		NeedVersionControl() bool
	}

	// Represents a source which has initiated a deployment.
	Source interface {
		Prepare(context.Context, App, any) (SourceData, error)      // Prepare the given payload for the given application, doing any needed validation
		Fetch(context.Context, DeploymentContext, Deployment) error // Retrieve deployment data and store them in the given path before passing in to a provider
	}
)
