package get_deployment

import (
	"time"

	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/storage"
)

var SourceDataTypes = storage.NewDiscriminatedMapper[SourceData]()

type (
	// Retrieve a deployment detail.
	Query struct {
		bus.Query[Deployment]

		AppID            string `json:"-"`
		DeploymentNumber int    `json:"-"`
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

	User struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	}

	Source struct {
		Discriminator string     `json:"discriminator"`
		Data          SourceData `json:"data"`
	}

	SourceData storage.Discriminated

	State struct {
		Status     uint8                  `json:"status"`
		Services   monad.Maybe[Services]  `json:"services"`
		ErrCode    monad.Maybe[string]    `json:"error_code"`
		StartedAt  monad.Maybe[time.Time] `json:"started_at"`
		FinishedAt monad.Maybe[time.Time] `json:"finished_at"`
	}

	Services []Service

	Service struct {
		Name  string              `json:"name"`
		Image string              `json:"image"`
		Url   monad.Maybe[string] `json:"url"`
	}
)

func (Query) Name_() string { return "deployment.query.get_deployment" }

func (s *Services) Scan(value any) error {
	return storage.ScanJSON(value, s)
}
