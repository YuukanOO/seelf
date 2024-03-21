package domain

import (
	"fmt"
	"strings"

	"github.com/YuukanOO/seelf/pkg/monad"
)

// Holds data related to the configuration of the final application. It should
// have everything needed to resolve service and image names and is the primarly used
// structure during the deployment by a provider.
type DeploymentConfig struct {
	appid       AppID
	appname     AppName
	environment Environment
	target      TargetID
	vars        monad.Maybe[ServicesEnv]
}

// Builds a new config snapshot for the given environment.
func (a *App) ConfigSnapshotFor(env Environment) (DeploymentConfig, error) {
	var (
		conf     EnvironmentConfig
		snapshot DeploymentConfig
	)

	switch env {
	case Production:
		conf = a.production
	case Staging:
		conf = a.staging
	default:
		return snapshot, ErrInvalidEnvironmentName
	}

	snapshot.appid = a.id
	snapshot.appname = a.name
	snapshot.environment = env
	snapshot.target = conf.Target()
	snapshot.vars = conf.Vars()

	return snapshot, nil
}

func (c DeploymentConfig) AppID() AppID                   { return c.appid }
func (c DeploymentConfig) AppName() AppName               { return c.appname }
func (c DeploymentConfig) Environment() Environment       { return c.environment }
func (c DeploymentConfig) Target() TargetID               { return c.target }
func (c DeploymentConfig) Vars() monad.Maybe[ServicesEnv] { return c.vars } // FIXME: If I want to follow my mantra, it should returns a readonly map

// Retrieve environment variables associated with the given service name.
// FIXME: If I want to follow my mantra, it should returns a readonly map
func (c DeploymentConfig) EnvironmentVariablesFor(service string) (m monad.Maybe[EnvVars]) {
	env, isSet := c.vars.TryGet()

	if !isSet {
		return m
	}

	vars, exists := env[service]

	if !exists {
		return m
	}

	m.Set(vars)

	return m
}

// Returns the subdomain that will be used to expose services of an app.
func (c DeploymentConfig) SubDomain() string {
	if c.environment.IsProduction() {
		return string(c.appname)

	}

	return fmt.Sprintf("%s-%s", c.appname, c.environment)
}

// Builds a unique image name for the given service.
func (c DeploymentConfig) ImageName(service string) string {
	return fmt.Sprintf("%s-%s/%s:%s", c.appname, strings.ToLower(string(c.appid)), service, c.environment)
}

// Builds a qualified name, truly unique, for the given service.
func (c DeploymentConfig) QualifiedName(service string) string {
	return fmt.Sprintf("%s-%s", c.ProjectName(), service)
}

// Retrieve the name of the project wich is the combination of the appname, environment and appid
// targeted by this configuration.
func (c DeploymentConfig) ProjectName() string {
	return fmt.Sprintf("%s-%s-%s", c.appname, strings.ToLower(string(c.appid)), c.environment)
}
