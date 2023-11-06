package domain

import (
	"fmt"

	"github.com/YuukanOO/seelf/pkg/monad"
)

// Holds data related to the configuration of the final application. It should
// have everything needed to resolve service and image name and is the primarly used
// structure during the deployment configuration process by the backend.
type Config struct {
	appname     UniqueAppName
	environment Environment
	env         monad.Maybe[ServicesEnv]
}

// Builds a new config from the given application.
func NewConfig(app App, environment Environment) Config {
	return Config{
		appname:     app.name,
		environment: environment,
		env:         app.envFor(environment),
	}
}

func (c Config) AppName() UniqueAppName        { return c.appname }
func (c Config) Environment() Environment      { return c.environment }
func (c Config) Env() monad.Maybe[ServicesEnv] { return c.env } // FIXME: If I want to follow my mantra, it should returns a readonly map

// Retrieve environment variables associated with the given service name.
// FIXME: If I want to follow my mantra, it should returns a readonly map
func (c Config) EnvironmentVariablesFor(service string) (m monad.Maybe[EnvVars]) {
	if !c.env.HasValue() {
		return m
	}

	vars, exists := c.env.MustGet()[service]

	if !exists {
		return m
	}

	return m.WithValue(vars)
}

// Returns the subdomain that will be used to expose services of an app.
func (c Config) SubDomain() string {
	if c.environment.IsProduction() {
		return string(c.appname)
	}

	return c.ProjectName()
}

// Retrieve the name of the project wich is the combination of the appname and the environment
// targeted by this configuration.
func (c Config) ProjectName() string {
	return fmt.Sprintf("%s-%s", c.appname, c.environment)
}
