package domain

import (
	"context"
	"time"

	"github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/bus"
	shared "github.com/YuukanOO/seelf/pkg/domain"
	"github.com/YuukanOO/seelf/pkg/event"
	"github.com/YuukanOO/seelf/pkg/id"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/must"
	"github.com/YuukanOO/seelf/pkg/storage"
)

var (
	ErrAppNameAlreadyTaken         = apperr.New("app_name_already_taken")
	ErrVersionControlNotConfigured = apperr.New("version_control_not_configured")
	ErrAppCleanupRequested         = apperr.New("app_cleanup_requested")
	ErrAppTargetChanged            = apperr.New("app_target_changed")
)

type (
	AppID           string
	HasAppsOnTarget bool

	App struct {
		event.Emitter

		id               AppID
		name             AppName
		versionControl   monad.Maybe[VersionControl]
		production       Environment
		staging          Environment
		cleanupRequested monad.Maybe[shared.Action[domain.UserID]]
		created          shared.Action[domain.UserID]
	}

	AppsReader interface {
		// Check if the naming is available (not use by another application with the same name on the same targets).
		CheckAppNamingAvailability(
			ctx context.Context,
			name AppName,
			production EnvironmentConfig,
			staging EnvironmentConfig,
		) (EnvironmentConfigRequirement, EnvironmentConfigRequirement, error)
		// Same as CheckAppNamingAvailability but used when updating the environment configuration with optional targets.
		CheckAppNamingAvailabilityByID(
			ctx context.Context,
			id AppID,
			production monad.Maybe[EnvironmentConfig],
			staging monad.Maybe[EnvironmentConfig],
		) (EnvironmentConfigRequirement, EnvironmentConfigRequirement, error)
		// Check if a specific target is used by an application.
		HasAppsOnTarget(context.Context, TargetID) (HasAppsOnTarget, error)
		GetByID(context.Context, AppID) (App, error)
	}

	AppsWriter interface {
		Write(context.Context, ...*App) error
	}

	AppCreated struct {
		bus.Notification

		ID         AppID
		Name       AppName
		Production Environment
		Staging    Environment
		Created    shared.Action[domain.UserID]
	}

	AppEnvChanged struct {
		bus.Notification

		ID          AppID
		Environment EnvironmentName
		Config      Environment
	}

	AppEnvMigrationStarted struct {
		bus.Notification

		ID          AppID
		Environment EnvironmentName
		Migration   EnvironmentMigration
	}

	AppEnvCleanedUp struct {
		bus.Notification

		ID          AppID
		Environment EnvironmentName
		Target      TargetID
		Config      Environment
	}

	AppVersionControlChanged struct {
		bus.Notification

		ID     AppID
		Config monad.Maybe[VersionControl]
	}

	AppCleanupRequested struct {
		bus.Notification

		ID         AppID
		Production Environment
		Staging    Environment
		Requested  shared.Action[domain.UserID]
	}

	AppDeleted struct {
		bus.Notification

		ID AppID
	}
)

func (AppCreated) Name_() string             { return "deployment.event.app_created" }
func (AppEnvChanged) Name_() string          { return "deployment.event.app_env_changed" }
func (AppEnvMigrationStarted) Name_() string { return "deployment.event.app_env_migration_started" }
func (AppEnvCleanedUp) Name_() string        { return "deployment.event.app_env_cleaned_up" }
func (AppVersionControlChanged) Name_() string {
	return "deployment.event.app_version_control_changed"
}
func (AppCleanupRequested) Name_() string { return "deployment.event.app_cleanup_requested" }
func (AppDeleted) Name_() string          { return "deployment.event.app_deleted" }

// Instantiates a new App.
func NewApp(
	name AppName,
	productionRequirement EnvironmentConfigRequirement,
	stagingRequirement EnvironmentConfigRequirement,
	createdBy domain.UserID,
) (app App, err error) {
	production, err := productionRequirement.Met()

	if err != nil {
		return app, err
	}

	staging, err := stagingRequirement.Met()

	if err != nil {
		return app, err
	}

	app.apply(AppCreated{
		ID:         id.New[AppID](),
		Name:       name,
		Production: newEnvironment(production),
		Staging:    newEnvironment(staging),
		Created:    shared.NewAction(createdBy),
	})

	return app, nil
}

// Recreates an app from the persistent storage.
func AppFrom(scanner storage.Scanner) (a App, err error) {
	var (
		version            event.Version
		url                monad.Maybe[Url]
		token              monad.Maybe[string]
		createdAt          time.Time
		createdBy          domain.UserID
		cleanupRequestedAt monad.Maybe[time.Time]
		cleanupRequestedBy monad.Maybe[string]

		productionMigrationTarget monad.Maybe[string]
		productionMigrationFrom   monad.Maybe[time.Time]
		productionMigrationTo     monad.Maybe[time.Time]

		stagingMigrationTarget monad.Maybe[string]
		stagingMigrationFrom   monad.Maybe[time.Time]
		stagingMigrationTo     monad.Maybe[time.Time]
	)

	if err = scanner.Scan(
		&a.id,
		&a.name,
		&url,
		&token,
		&productionMigrationTarget,
		&productionMigrationFrom,
		&productionMigrationTo,
		&a.production.since,
		&a.production.cleaned,
		&a.production.config.target,
		&a.production.config.vars,
		&stagingMigrationTarget,
		&stagingMigrationFrom,
		&stagingMigrationTo,
		&a.staging.since,
		&a.staging.cleaned,
		&a.staging.config.target,
		&a.staging.config.vars,
		&createdAt,
		&createdBy,
		&cleanupRequestedAt,
		&cleanupRequestedBy,
		&version,
	); err != nil {
		return a, err
	}

	event.Hydrate(&a, version)

	a.created = shared.ActionFrom(createdBy, createdAt)

	if requestedAt, isSet := cleanupRequestedAt.TryGet(); isSet {
		a.cleanupRequested.Set(
			shared.ActionFrom(domain.UserID(cleanupRequestedBy.MustGet()), requestedAt),
		)
	}

	if prodMigrationTarget, hasProdMigration := productionMigrationTarget.TryGet(); hasProdMigration {
		a.production.migration.Set(EnvironmentMigration{
			target:   TargetID(prodMigrationTarget),
			interval: must.Panic(shared.NewTimeInterval(productionMigrationFrom.MustGet(), productionMigrationTo.MustGet())),
		})
	}

	if stagingMigrationTarget, hasStagingMigration := stagingMigrationTarget.TryGet(); hasStagingMigration {
		a.staging.migration.Set(EnvironmentMigration{
			target:   TargetID(stagingMigrationTarget),
			interval: must.Panic(shared.NewTimeInterval(stagingMigrationFrom.MustGet(), stagingMigrationTo.MustGet())),
		})
	}

	// vcs url has been set, reconstitute the vcs config
	if u, isSet := url.TryGet(); isSet {
		a.versionControl.Set(VersionControl{
			url:   u,
			token: token,
		})
	}

	return a, err
}

// Sets an app version control configuration.
func (a *App) UseVersionControl(config VersionControl) error {
	if a.cleanupRequested.HasValue() {
		return ErrAppCleanupRequested
	}

	if existing, isSet := a.versionControl.TryGet(); isSet && config == existing {
		return nil
	}

	a.apply(AppVersionControlChanged{
		ID:     a.id,
		Config: monad.Value(config),
	})

	return nil
}

// Removes the version control configuration from the app.
func (a *App) RemoveVersionControl() error {
	if a.cleanupRequested.HasValue() {
		return ErrAppCleanupRequested
	}

	if !a.versionControl.HasValue() {
		return nil
	}

	a.apply(AppVersionControlChanged{
		ID: a.id,
	})

	return nil
}

// Updates the production configuration for this application.
func (a *App) HasProductionConfig(configRequirement EnvironmentConfigRequirement) error {
	return a.tryUpdateEnvironmentConfig(Production, a.production, configRequirement)
}

// Updates the staging configuration for this application.
func (a *App) HasStagingConfig(configRequirement EnvironmentConfigRequirement) error {
	return a.tryUpdateEnvironmentConfig(Staging, a.staging, configRequirement)
}

// Request application deletion meaning the application resources should be removed
// and the application deleted when every resources are freed.
func (a *App) RequestDelete(requestedBy domain.UserID) error {
	if a.cleanupRequested.HasValue() {
		return ErrAppCleanupRequested
	}

	a.apply(AppCleanupRequested{
		ID:         a.id,
		Production: a.production,
		Staging:    a.staging,
		Requested:  shared.NewAction(requestedBy),
	})

	return nil
}

// Marks the application has being cleaned for a specific environment and a specific target.
func (a *App) CleanedUp(environment EnvironmentName, target TargetID) error {
	env, err := a.environmentFor(environment)

	if err != nil {
		return err
	}

	if err := env.clean(target, a.cleanupRequested.HasValue()); err != nil {
		return err
	}

	a.apply(AppEnvCleanedUp{
		ID:          a.id,
		Environment: environment,
		Target:      target,
		Config:      env,
	})

	if a.production.isFullyCleaned() && a.staging.isFullyCleaned() {
		a.apply(AppDeleted{
			ID: a.id,
		})
	}

	return nil
}

func (a *App) ID() AppID                                   { return a.id }
func (a *App) VersionControl() monad.Maybe[VersionControl] { return a.versionControl }

func (a *App) environmentFor(name EnvironmentName) (Environment, error) {
	switch name {
	case Production:
		return a.production, nil
	case Staging:
		return a.staging, nil
	default:
		return Environment{}, ErrInvalidEnvironmentName
	}
}

func (a *App) tryUpdateEnvironmentConfig(
	name EnvironmentName,
	environment Environment,
	updatedConfigRequirement EnvironmentConfigRequirement,
) error {
	if a.cleanupRequested.HasValue() {
		return ErrAppCleanupRequested
	}

	updatedConfig, err := updatedConfigRequirement.Met()

	if err != nil {
		return err
	}

	migration, updated, err := environment.update(updatedConfig)

	if err != nil || !updated {
		return err
	}

	a.apply(AppEnvChanged{
		ID:          a.id,
		Environment: name,
		Config:      environment,
	})

	if m, migrationNeeded := migration.TryGet(); migrationNeeded {
		a.apply(AppEnvMigrationStarted{
			ID:          a.id,
			Environment: name,
			Migration:   m,
		})
	}

	return nil
}

func (a *App) setEnvironmentFor(name EnvironmentName, config Environment) {
	switch name {
	case Production:
		a.production = config
	case Staging:
		a.staging = config
	}
}

func (a *App) apply(e event.Event) {
	switch evt := e.(type) {
	case AppCreated:
		a.id = evt.ID
		a.name = evt.Name
		a.production = evt.Production
		a.staging = evt.Staging
		a.created = evt.Created
	case AppEnvChanged:
		a.setEnvironmentFor(evt.Environment, evt.Config)
	case AppEnvCleanedUp:
		a.setEnvironmentFor(evt.Environment, evt.Config)
	case AppVersionControlChanged:
		a.versionControl = evt.Config
	case AppCleanupRequested:
		a.cleanupRequested.Set(evt.Requested)
	}

	event.Store(a, e)
}
