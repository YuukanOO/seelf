package get_target

import (
	"time"

	"github.com/YuukanOO/seelf/internal/deployment/app"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/storage"
)

var ProviderConfigTypes = storage.NewDiscriminatedMapper(func(pc ProviderConfig) string { return pc.Kind() })

type (
	// Retrieve one target
	Query struct {
		bus.Query[Target]

		ID string `json:"id"`
	}

	Target struct {
		ID                 string                       `json:"id"`
		Name               string                       `json:"name"`
		Url                monad.Maybe[string]          `json:"url"`
		Provider           Provider                     `json:"provider"`
		State              State                        `json:"state"`
		CleanupRequestedAt monad.Maybe[time.Time]       `json:"cleanup_requested_at"`
		CleanupRequestedBy monad.Maybe[app.UserSummary] `json:"cleanup_requested_by"`
		CreatedAt          time.Time                    `json:"created_at"`
		CreatedBy          app.UserSummary              `json:"created_by"`
	}

	State struct {
		Status           uint8                  `json:"status"`
		ErrCode          monad.Maybe[string]    `json:"error_code"`
		LastReadyVersion monad.Maybe[time.Time] `json:"last_ready_version"`
	}

	Provider struct {
		Kind string         `json:"kind"`
		Data ProviderConfig `json:"data"`
	}

	ProviderConfig interface {
		Kind() string
	}
)

func (Query) Name_() string { return "deployment.query.get_target" }
