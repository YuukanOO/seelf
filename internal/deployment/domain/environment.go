package domain

import (
	"database/sql/driver"
	"reflect"
	"time"

	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/storage"
)

var (
	ErrInvalidEnvironmentName = apperr.New("invalid_environment_name")
)

const (
	// The production environment has a special meaning when determining the application domain.
	Production Environment = "production"
	// Staging environment
	Staging Environment = "staging"
)

type (
	Environment string             // Represents a valid environment name
	EnvVars     map[string]string  // Environment variables key pair
	ServicesEnv map[string]EnvVars // Environment variables per service name

	// Represents a specific environment configuration.
	// The version field is used during the cleanup process to check for successfull deployments
	// during a specific interval (the last target change).
	EnvironmentConfig struct {
		target  TargetID
		version time.Time
		vars    monad.Maybe[ServicesEnv]
	}
)

// Creates a new environment value object from a raw value.
func EnvironmentFrom(value string) (Environment, error) {
	switch Environment(value) {
	case Production:
		return Production, nil
	case Staging:
		return Staging, nil
	default:
		return "", ErrInvalidEnvironmentName
	}
}

// Returns true if the given environment represents the production one.
func (e Environment) IsProduction() bool { return e == Production }

// Builds a new environment config targetting the specificied target.
func NewEnvironmentConfig(target TargetID) EnvironmentConfig {
	return EnvironmentConfig{
		target:  target,
		version: time.Now().UTC(),
	}
}

// Add the given environment variables per service to this configuration.
func (e *EnvironmentConfig) HasEnvironmentVariables(vars ServicesEnv) {
	e.vars.Set(vars)
}

// Check if two environment config are equals, does not compare version.
func (e EnvironmentConfig) Equals(other EnvironmentConfig) bool {
	return e.target == other.target && reflect.DeepEqual(e.vars, other.vars)
}

func (e EnvironmentConfig) Target() TargetID               { return e.target }
func (e EnvironmentConfig) Version() time.Time             { return e.version }
func (e EnvironmentConfig) Vars() monad.Maybe[ServicesEnv] { return e.vars }

// Builds the map of services variables from a raw value.
func ServicesEnvFrom(raw map[string]map[string]string) ServicesEnv {
	result := make(ServicesEnv, len(raw))

	for service, vars := range raw {
		if vars == nil {
			continue
		}

		result[service] = vars
	}

	return result
}

func (e ServicesEnv) Value() (driver.Value, error) { return storage.ValueJSON(e) }
func (e *ServicesEnv) Scan(value any) error        { return storage.ScanJSON(value, e) }
