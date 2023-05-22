package domain

import (
	"database/sql/driver"
	"encoding/json"
	"reflect"
	"regexp"

	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/storage"
)

var (
	ErrInvalidEnvironmentName = apperr.New("invalid_environment_name")
	allowedEnvironmentRe      = regexp.MustCompile("^(production|staging)$") // For now, limit to production / staging
)

const (
	// The production environment has a special meaning when determining the application domain.
	Production Environment = "production"
	// Staging environment
	Staging Environment = "staging"
)

type Environment string

// Creates a new environment value object from a raw value.
func EnvironmentFrom(value string) (Environment, error) {
	if !allowedEnvironmentRe.MatchString(value) {
		return "", ErrInvalidEnvironmentName
	}

	return Environment(value), nil
}

// Returns true if the given environment represents the production one.
func (e Environment) IsProduction() bool { return e == Production }

type (
	EnvVars         map[string]string           // Environment variables key pair
	ServicesEnv     map[string]EnvVars          // Environment variables per service name
	EnvironmentsEnv map[Environment]ServicesEnv // Environment variables per deployment environment
)

// Builds the map of environment variables per env and per service from a raw value.
func EnvironmentsEnvFrom(raw map[string]map[string]map[string]string) (EnvironmentsEnv, error) {
	result := EnvironmentsEnv{}

	for envname, services := range raw {
		env, err := EnvironmentFrom(envname)

		if err != nil {
			return EnvironmentsEnv{}, err
		}

		servicesEnv := ServicesEnv{}

		for service, vars := range services {
			servicesEnv[service] = vars
		}

		result[env] = servicesEnv
	}

	return result, nil
}

func (e EnvironmentsEnv) Equals(other EnvironmentsEnv) bool {
	return reflect.DeepEqual(e, other) // Using DeepEqual here is much more simpler
}

func (e EnvironmentsEnv) Value() (driver.Value, error) {
	r, err := json.Marshal(e)

	return string(r), err
}

func (e *EnvironmentsEnv) Scan(value any) error {
	return storage.ScanJSON(value, &e)
}

func (e ServicesEnv) Value() (driver.Value, error) {
	r, err := json.Marshal(e)

	return string(r), err
}

func (e *ServicesEnv) Scan(value any) error {
	return storage.ScanJSON(value, &e)
}
