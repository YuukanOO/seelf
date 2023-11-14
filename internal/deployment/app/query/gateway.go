package query

import (
	"context"
	"time"

	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/query"
	"github.com/YuukanOO/seelf/pkg/storage"
)

var SourceDataTypes = storage.NewDiscriminatedMapper[SourceData]()

type (
	// Access to the underlying storage adapter for read use cases
	Gateway interface {
		GetAppByID(context.Context, string) (AppDetail, error)
		GetAllApps(context.Context) ([]App, error)
		GetAllDeploymentsByApp(context.Context, string, GetDeploymentsFilters) (query.Paginated[Deployment], error)
		GetDeploymentByID(context.Context, string, int) (Deployment, error)
	}

	GetDeploymentsFilters struct {
		Page        monad.Maybe[int]    `form:"page"`
		Environment monad.Maybe[string] `form:"environment"`
	}

	App struct {
		ID                 string                 `json:"id"`
		Name               string                 `json:"name"`
		CleanupRequestedAt monad.Maybe[time.Time] `json:"cleanup_requested_at"`
		CreatedAt          time.Time              `json:"created_at"`
		CreatedBy          User                   `json:"created_by"`
		Environments       map[string]Deployment  `json:"environments"`
	}

	AppDetail struct {
		App
		Env monad.Maybe[Env]       `json:"env"`
		VCS monad.Maybe[VCSConfig] `json:"vcs"`
	}

	Deployment struct {
		AppID            string    `json:"app_id"`
		DeploymentNumber int       `json:"deployment_number"`
		Environment      string    `json:"environment"`
		Source           Source    `json:"source"`
		State            State     `json:"state"`
		RequestedAt      time.Time `json:"requested_at"`
		RequestedBy      User      `json:"requested_by"`
	}

	Source struct {
		Discriminator string     `json:"discriminator"`
		Data          SourceData `json:"data"`
	}

	SourceData storage.Discriminated

	VCSConfig struct {
		Url   string                          `json:"url"`
		Token monad.Maybe[query.SecretString] `json:"token"`
	}

	State struct {
		Status     uint8                  `json:"status"`
		Services   monad.Maybe[Services]  `json:"services"`
		ErrCode    monad.Maybe[string]    `json:"error_code"`
		StartedAt  monad.Maybe[time.Time] `json:"started_at"`
		FinishedAt monad.Maybe[time.Time] `json:"finished_at"`
	}

	Services []Service
	Env      map[string]map[string]map[string]string

	Service struct {
		Name  string              `json:"name"`
		Image string              `json:"image"`
		Url   monad.Maybe[string] `json:"url"`
	}

	User struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	}
)

func (s *Services) Scan(value any) error {
	return storage.ScanJSON(value, s)
}

func (e *Env) Scan(value any) error {
	return storage.ScanJSON(value, e)
}
