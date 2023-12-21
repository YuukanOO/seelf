package get_apps

import (
	"time"

	"github.com/YuukanOO/seelf/internal/deployment/app/get_deployment"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/monad"
)

type (
	// Retrieve all apps.
	Query struct {
		bus.Query[[]App]
	}

	App struct {
		ID                 string                               `json:"id"`
		Name               string                               `json:"name"`
		CleanupRequestedAt monad.Maybe[time.Time]               `json:"cleanup_requested_at"`
		CreatedAt          time.Time                            `json:"created_at"`
		CreatedBy          get_deployment.User                  `json:"created_by"`
		Environments       map[string]get_deployment.Deployment `json:"environments"`
	}
)

func (Query) Name_() string { return "deployment.query.get_apps" }
