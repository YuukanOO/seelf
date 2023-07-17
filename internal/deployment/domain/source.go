package domain

import (
	"context"

	"github.com/YuukanOO/seelf/pkg/apperr"
)

var (
	ErrNoValidSourceFound   = apperr.New("no_valid_source_found")
	ErrInvalidSourcePayload = apperr.New("invalid_source_payload")
	ErrSourceFetchFailed    = apperr.New("source_fetch_failed")
)

// Represents a source which has initiated a deployment.
type Source interface {
	Prepare(App, any) (Meta, error)          // Prepare the given payload for the given application, doing any needed validation
	Fetch(context.Context, Deployment) error // Retrieve deployment data before passing in to a backend
}
