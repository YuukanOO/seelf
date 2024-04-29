package domain

import (
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

// Returns the subdomain that will be used to expose a specific service.
func (c DeploymentConfig) SubDomain(service string, isDefault bool) string {
	subdomain := string(c.appname)

	if !c.environment.IsProduction() {
		subdomain += "-" + string(c.environment)
	}

	// If the default domain has already been taken by another service, build a
	// unique subdomain with the service name being exposed.
	if !isDefault {
		subdomain = service + "." + subdomain
	}

	return subdomain
}

// Builds a unique image name for the given service.
func (c DeploymentConfig) ImageName(service string) string {
	return string(c.appname) + "-" + strings.ToLower(string(c.appid)) + "/" + service + ":" + string(c.environment)
}

// Builds a qualified name, truly unique, for the given service.
func (c DeploymentConfig) QualifiedName(service string) string {
	return c.ProjectName() + "-" + service
}

// Retrieve the name of the project wich is the combination of the appname, environment and appid
// targeted by this configuration.
func (c DeploymentConfig) ProjectName() string {
	return string(c.appname) + "-" + string(c.environment) + "-" + strings.ToLower(string(c.appid))
}
