package get_app_deployments

import (
	"github.com/YuukanOO/seelf/internal/deployment/app/get_deployment"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/storage"
)

// Retrieve all deployments for an app.
type Query struct {
	bus.Query[storage.Paginated[get_deployment.Deployment]]

	AppID       string              `json:"-"`
	Page        monad.Maybe[int]    `form:"page"`
	Environment monad.Maybe[string] `form:"environment"`
}

func (Query) Name_() string { return "deployment.query.get_app_deployments" }
