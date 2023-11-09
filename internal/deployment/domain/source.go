package domain

import (
	"context"
	"errors"

	"github.com/YuukanOO/seelf/pkg/storage"
)

var (
	ErrNoValidSourceFound   = errors.New("no_valid_source_found")
	ErrInvalidSourcePayload = errors.New("invalid_source_payload")
	ErrSourceFetchFailed    = errors.New("source_fetch_failed")

	SourceDataTypes = storage.NewDiscriminatedMapper[SourceData]()
)

type (
	// Contains stuff related to how the deployment has been triggered.
	// The inner data depends on the Source which has been requested.
	SourceData interface {
		storage.Discriminated
		NeedVCS() bool
	}

	// Represents a source which has initiated a deployment.
	Source interface {
		Prepare(App, any) (SourceData, error)    // Prepare the given payload for the given application, doing any needed validation
		Fetch(context.Context, Deployment) error // Retrieve deployment data before passing in to a backend
	}
)
