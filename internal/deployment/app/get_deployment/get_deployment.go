package get_deployment

import (
	"time"

	"github.com/YuukanOO/seelf/internal/deployment/app"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/storage"
)

var SourceDataTypes = storage.NewDiscriminatedMapper(func(sd SourceData) string { return sd.Kind() })

type (
	// Retrieve a deployment detail.
	Query struct {
		bus.Query[Deployment]

		AppID            string `json:"-"`
		DeploymentNumber int    `json:"-"`
	}

	Deployment struct {
		AppID            string          `json:"app_id"`
		DeploymentNumber int             `json:"deployment_number"`
		Environment      string          `json:"environment"`
		Target           TargetSummary   `json:"target"`
		Source           Source          `json:"source"`
		State            State           `json:"state"`
		RequestedAt      time.Time       `json:"requested_at"`
		RequestedBy      app.UserSummary `json:"requested_by"`
	}

	TargetSummary struct {
		ID   string              `json:"id"`
		Name monad.Maybe[string] `json:"name"` // Since the target could have been deleted, the name is nullable here.
		Url  monad.Maybe[string] `json:"url"`
	}

	Source struct {
		Discriminator string     `json:"discriminator"`
		Data          SourceData `json:"data"`
	}

	SourceData interface {
		Kind() string
	}

	State struct {
		Status     uint8                  `json:"status"`
		Services   monad.Maybe[Services]  `json:"services"`
		ErrCode    monad.Maybe[string]    `json:"error_code"`
		StartedAt  monad.Maybe[time.Time] `json:"started_at"`
		FinishedAt monad.Maybe[time.Time] `json:"finished_at"`
	}

	Services []Service

	Service struct {
		Name      string              `json:"name"`
		Image     string              `json:"image"`
		Subdomain monad.Maybe[string] `json:"subdomain"`
		Url       monad.Maybe[string] `json:"url"`
	}
)

func (Query) Name_() string { return "deployment.query.get_deployment" }

func (s *Services) Scan(value any) error {
	return storage.ScanJSON(value, s)
}
