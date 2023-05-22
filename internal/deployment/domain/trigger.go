package domain

import (
	"context"

	"github.com/YuukanOO/seelf/pkg/apperr"
)

var (
	ErrNoValidTriggerFound   = apperr.New("no_valid_trigger_found")
	ErrInvalidTriggerPayload = apperr.New("invalid_trigger_payload")
	ErrTriggerFetchFailed    = apperr.New("trigger_fetch_failed")
)

// Represents a trigger which has initiated a deployment.
type Trigger interface {
	Prepare(App, any) (Meta, error)          // Prepare the given payload for the given application, doing any needed validation
	Fetch(context.Context, Deployment) error // Retrieve deployment data before passing in to a backend
}
