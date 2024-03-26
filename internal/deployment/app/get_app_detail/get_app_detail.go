package get_app_detail

import (
	"github.com/YuukanOO/seelf/internal/deployment/app/get_apps"
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
		get_apps.App

		Production Environment            `json:"production"`
		Staging    Environment            `json:"staging"`
		VCS        monad.Maybe[VCSConfig] `json:"vcs"`
	}

	VCSConfig struct {
		Url   string                            `json:"url"`
		Token monad.Maybe[storage.SecretString] `json:"token"`
	}

	Environment struct {
		Target TargetSummary            `json:"target"`
		Vars   monad.Maybe[ServicesEnv] `json:"vars"`
	}

	TargetSummary struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Url  string `json:"url"`
	}

	ServicesEnv map[string]map[string]string
)

func (Query) Name_() string { return "deployment.query.get_app_detail" }

func (e *ServicesEnv) Scan(value any) error {
	return storage.ScanJSON(value, e)
}
