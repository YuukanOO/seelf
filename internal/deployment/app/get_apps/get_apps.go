package get_apps

import (
	"time"

	"github.com/YuukanOO/seelf/internal/deployment/app"
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
		ID                 string                       `json:"id"`
		Name               string                       `json:"name"`
		CleanupRequestedAt monad.Maybe[time.Time]       `json:"cleanup_requested_at"`
		CleanupRequestedBy monad.Maybe[app.UserSummary] `json:"cleanup_requested_by"`
		CreatedAt          time.Time                    `json:"created_at"`
		CreatedBy          app.UserSummary              `json:"created_by"`
		LatestDeployments  LatestDeployments            `json:"latest_deployments"`
		ProductionTarget   app.TargetSummary            `json:"production_target"`
		StagingTarget      app.TargetSummary            `json:"staging_target"`
	}

	LatestDeployments struct {
		Production monad.Maybe[get_deployment.Deployment] `json:"production"`
		Staging    monad.Maybe[get_deployment.Deployment] `json:"staging"`
	}
)

func (Query) Name_() string { return "deployment.query.get_apps" }
