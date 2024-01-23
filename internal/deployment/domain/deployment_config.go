package domain

import (
	"fmt"

	"github.com/YuukanOO/seelf/pkg/monad"
)

// Holds data related to the configuration of the final application. It should
// have everything needed to resolve service and image names and is the primarly used
// structure during the deployment by a provider.
type DeploymentConfig struct {
	appname     AppName
	environment Environment
	target      TargetID
	vars        monad.Maybe[ServicesEnv]
}

// Builds a new config snapshot for the given environment.
func (a App) ConfigSnapshotFor(env Environment) (DeploymentConfig, error) {
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

	snapshot.appname = a.name
	snapshot.environment = env
	snapshot.target = conf.Target()
	snapshot.vars = conf.Vars()

	return snapshot, nil
}

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

	return m.WithValue(vars)
}

// Returns the subdomain that will be used to expose services of an app.
func (c DeploymentConfig) SubDomain() string {
	if c.environment.IsProduction() {
		return string(c.appname)
	}

	return c.ProjectName()
}

// Retrieve the name of the project wich is the combination of the appname and the environment
// targeted by this configuration.
func (c DeploymentConfig) ProjectName() string {
	return fmt.Sprintf("%s-%s", c.appname, c.environment)
}
