package get_app_detail

import (
	"time"

	"github.com/YuukanOO/seelf/internal/deployment/app"
	"github.com/YuukanOO/seelf/internal/deployment/app/get_deployment"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/storage"
)

type (
	// Retrieve an app detail.
	Query struct {
		bus.Query[App]

		ID string `json:"-"`
	}

	App struct {
		ID                 string                                           `json:"id"`
		Name               string                                           `json:"name"`
		CleanupRequestedAt monad.Maybe[time.Time]                           `json:"cleanup_requested_at"`
		CleanupRequestedBy monad.Maybe[app.UserSummary]                     `json:"cleanup_requested_by"`
		CreatedAt          time.Time                                        `json:"created_at"`
		CreatedBy          app.UserSummary                                  `json:"created_by"`
		LatestDeployments  app.LatestDeployments[get_deployment.Deployment] `json:"latest_deployments"`
		Production         EnvironmentConfig                                `json:"production"`
		Staging            EnvironmentConfig                                `json:"staging"`
		VersionControl     monad.Maybe[VersionControl]                      `json:"version_control"`
	}

	VersionControl struct {
		Url   string                            `json:"url"`
		Token monad.Maybe[storage.SecretString] `json:"token"`
	}

	EnvironmentConfig struct {
		Target app.TargetSummary        `json:"target"`
		Vars   monad.Maybe[ServicesEnv] `json:"vars"`
	}

	ServicesEnv map[string]map[string]string
)

func (Query) Name_() string { return "deployment.query.get_app_detail" }

func (e *ServicesEnv) Scan(value any) error {
	return storage.ScanJSON(value, e)
}
