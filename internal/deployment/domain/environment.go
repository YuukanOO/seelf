package domain

import (
	"database/sql/driver"
	"reflect"
	"time"

	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/domain"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/storage"
)

var (
	ErrInvalidEnvironmentName                = apperr.New("invalid_environment_name")
	ErrAppEnvironmentMigrationAlreadyRunning = apperr.New("app_environment_migration_already_running")
	ErrAppEnvironmentAlreadyCleaned          = apperr.New("app_environment_already_cleaned")
	ErrAppEnvironmentCleanupNotAllowed       = apperr.New("app_environment_cleanup_not_allowed")
	ErrAppEnvironmentTargetInvalid           = apperr.New("app_environment_target_invalid")
)

const (
	// The production environment has a special meaning when determining the application domain.
	Production EnvironmentName = "production"
	// Staging environment
	Staging EnvironmentName = "staging"
)

// Represents the whole environment for a particular application managing
// the config update and the environment migration if needed.
type Environment struct {
	migration monad.Maybe[EnvironmentMigration]
	since     time.Time
	cleaned   bool
	config    EnvironmentConfig
}

// Builds a new environment from given configuration.
func newEnvironment(config EnvironmentConfig) Environment {
	return Environment{
		since:  time.Now().UTC(),
		config: config,
	}
}

// Try to update the inner configuration for this environment.
// If a migration from a target to a new one should occurs, it will returns said
// needed migration.
// Returns wether or not the config has been updated in the process.
func (e *Environment) update(config EnvironmentConfig) (monad.Maybe[EnvironmentMigration], bool, error) {
	var result monad.Maybe[EnvironmentMigration]

	if config.Equals(e.config) {
		return result, false, nil
	}

	var (
		initialTarget = e.config.target
		targetChanged = initialTarget != config.target
	)

	if targetChanged && e.migration.HasValue() {
		return result, false, ErrAppEnvironmentMigrationAlreadyRunning
	}

	e.config = config

	if !targetChanged {
		return result, true, nil
	}

	now := time.Now().UTC()
	interval, err := domain.NewTimeInterval(e.since, now)

	if err != nil {
		return result, false, err
	}

	result.Set(EnvironmentMigration{
		target:   initialTarget,
		interval: interval,
	})

	e.since = now
	e.migration = result

	return result, true, nil
}

func (e *Environment) clean(target TargetID, allowCurrentTargetCleanup bool) error {
	if target == e.config.target {
		if !allowCurrentTargetCleanup {
			return ErrAppEnvironmentCleanupNotAllowed
		}

		if e.cleaned {
			return ErrAppEnvironmentAlreadyCleaned
		}

		e.cleaned = true
		return nil
	}

	if migration, hasMigration := e.migration.TryGet(); hasMigration && target == migration.target {
		e.migration.Unset()
		return nil
	}

	return ErrAppEnvironmentTargetInvalid
}

func (e Environment) Since() time.Time                             { return e.since }
func (e Environment) Migration() monad.Maybe[EnvironmentMigration] { return e.migration }
func (e Environment) IsCleanedUp() bool                            { return e.cleaned }
func (e Environment) Config() EnvironmentConfig                    { return e.config }
func (e Environment) isFullyCleaned() bool                         { return !e.migration.HasValue() && e.cleaned }

// Represents an app migration from a target to another one.
type EnvironmentMigration struct {
	target   TargetID
	interval domain.TimeInterval
}

func (m EnvironmentMigration) Target() TargetID              { return m.target }
func (m EnvironmentMigration) Interval() domain.TimeInterval { return m.interval }

type EnvironmentName string // Represents a valid environment name

// Creates a new environment value object from a raw value.
func EnvironmentNameFrom(value string) (EnvironmentName, error) {
	switch EnvironmentName(value) {
	case Production:
		return Production, nil
	case Staging:
		return Staging, nil
	default:
		return "", ErrInvalidEnvironmentName
	}
}

// Returns true if the given environment represents the production one.
func (e EnvironmentName) IsProduction() bool { return e == Production }

// Represents a specific environment configuration.
type EnvironmentConfig struct {
	target TargetID
	vars   monad.Maybe[ServicesEnv]
}

// Builds a new environment config targetting the specificied target.
func NewEnvironmentConfig(target TargetID) EnvironmentConfig {
	return EnvironmentConfig{
		target: target,
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
func (e EnvironmentConfig) Vars() monad.Maybe[ServicesEnv] { return e.vars }

type (
	EnvVars     map[string]string  // Environment variables key pair
	ServicesEnv map[string]EnvVars // Environment variables per service name
)

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
