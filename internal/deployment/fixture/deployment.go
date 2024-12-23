//go:build !release

package fixture

import (
	"database/sql/driver"

	auth "github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/id"
	"github.com/YuukanOO/seelf/pkg/must"
	"github.com/YuukanOO/seelf/pkg/storage"
)

type (
	deploymentOption struct {
		uid         auth.UserID
		environment domain.EnvironmentName
		source      domain.SourceData
		app         domain.App
	}

	DeploymentOptionBuilder func(*deploymentOption)
)

func Deployment(options ...DeploymentOptionBuilder) domain.Deployment {
	opts := deploymentOption{
		uid:         id.New[auth.UserID](),
		environment: domain.Production,
		source:      SourceData(),
		app:         App(),
	}

	for _, o := range options {
		o(&opts)
	}

	return must.Panic(opts.app.NewDeployment(1, opts.source, opts.environment, opts.uid))
}

func FromApp(app domain.App) DeploymentOptionBuilder {
	return func(o *deploymentOption) {
		o.app = app
	}
}

func WithSourceData(source domain.SourceData) DeploymentOptionBuilder {
	return func(o *deploymentOption) {
		o.source = source
	}
}

func WithDeploymentRequestedBy(uid auth.UserID) DeploymentOptionBuilder {
	return func(o *deploymentOption) {
		o.uid = uid
	}
}

func ForEnvironment(environment domain.EnvironmentName) DeploymentOptionBuilder {
	return func(o *deploymentOption) {
		o.environment = environment
	}
}

type (
	sourceDataOption struct {
		UseVersionControl bool
	}

	SourceDataOptionBuilder func(*sourceDataOption)
)

func SourceData(options ...SourceDataOptionBuilder) domain.SourceData {
	var opts sourceDataOption

	for _, o := range options {
		o(&opts)
	}

	return opts
}

func (sourceDataOption) Kind() string                   { return "test" }
func (m sourceDataOption) NeedVersionControl() bool     { return m.UseVersionControl }
func (m sourceDataOption) Value() (driver.Value, error) { return storage.ValueJSON(m) }

func WithVersionControlNeeded() SourceDataOptionBuilder {
	return func(o *sourceDataOption) {
		o.UseVersionControl = true
	}
}

func init() {
	domain.SourceDataTypes.Register(sourceDataOption{}, func(s string) (domain.SourceData, error) {
		return storage.UnmarshalJSON[sourceDataOption](s)
	})
}
