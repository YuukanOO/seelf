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
	"github.com/YuukanOO/seelf/pkg/storage"
)

var (
	ErrAppNameAlreadyTaken         = apperr.New("app_name_already_taken")
	ErrVersionControlNotConfigured = apperr.New("version_control_not_configured")
	ErrAppCleanupNeeded            = apperr.New("app_cleanup_needed")
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
		production       EnvironmentConfig
		staging          EnvironmentConfig
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
		Production EnvironmentConfig
		Staging    EnvironmentConfig
		Created    shared.Action[domain.UserID]
	}

	AppEnvChanged struct {
		bus.Notification

		ID          AppID
		Environment Environment
		Config      EnvironmentConfig
		OldConfig   EnvironmentConfig // Old configuration, used to ease the cleanup handling
	}

	AppVersionControlConfigured struct {
		bus.Notification

		ID     AppID
		Config VersionControl
	}

	AppVersionControlRemoved struct {
		bus.Notification

		ID AppID
	}

	AppCleanupRequested struct {
		bus.Notification

		ID               AppID
		ProductionConfig EnvironmentConfig
		StagingConfig    EnvironmentConfig
		Requested        shared.Action[domain.UserID]
	}

	AppDeleted struct {
		bus.Notification

		ID AppID
	}
)

func (AppCreated) Name_() string    { return "deployment.event.app_created" }
func (AppEnvChanged) Name_() string { return "deployment.event.app_env_changed" }
func (AppVersionControlConfigured) Name_() string {
	return "deployment.event.app_version_control_configured"
}
func (AppVersionControlRemoved) Name_() string { return "deployment.event.app_version_control_removed" }
func (AppCleanupRequested) Name_() string      { return "deployment.event.app_cleanup_requested" }
func (AppDeleted) Name_() string               { return "deployment.event.app_deleted" }

func (e AppEnvChanged) TargetHasChanged() bool { return e.Config.Target() != e.OldConfig.Target() }

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
		Production: production,
		Staging:    staging,
		Created:    shared.NewAction(createdBy),
	})

	return app, nil
}

// Recreates an app from the persistent storage.
func AppFrom(scanner storage.Scanner) (a App, err error) {
	var (
		url                monad.Maybe[Url]
		token              monad.Maybe[string]
		createdAt          time.Time
		createdBy          domain.UserID
		cleanupRequestedAt monad.Maybe[time.Time]
		cleanupRequestedBy monad.Maybe[string]
	)

	err = scanner.Scan(
		&a.id,
		&a.name,
		&url,
		&token,
		&a.production.target,
		&a.production.version,
		&a.production.vars,
		&a.staging.target,
		&a.staging.version,
		&a.staging.vars,
		&cleanupRequestedAt,
		&cleanupRequestedBy,
		&createdAt,
		&createdBy,
	)

	a.created = shared.ActionFrom(createdBy, createdAt)

	if requestedAt, isSet := cleanupRequestedAt.TryGet(); isSet {
		a.cleanupRequested.Set(
			shared.ActionFrom(domain.UserID(cleanupRequestedBy.MustGet()), requestedAt),
		)
	}

	// vcs url has been set, reconstitute the vcs config
	if u, isSet := url.TryGet(); isSet {
		vcs := NewVersionControl(u)

		if tok, isSet := token.TryGet(); isSet {
			vcs.Authenticated(tok)
		}

		a.versionControl.Set(vcs)
	}

	return a, err
}

// Sets an app version control configuration.
func (a *App) UseVersionControl(config VersionControl) error {
	if a.cleanupRequested.HasValue() {
		return ErrAppCleanupRequested
	}

	if existing, isSet := a.versionControl.TryGet(); isSet && config.Equals(existing) {
		return nil
	}

	a.apply(AppVersionControlConfigured{
		ID:     a.id,
		Config: config,
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

	a.apply(AppVersionControlRemoved{
		ID: a.id,
	})

	return nil
}

// Updates the production configuration for this application.
func (a *App) HasProductionConfig(configRequirement EnvironmentConfigRequirement) error {
	return a.tryUpdateEnvironmentConfig(Production, configRequirement)
}

// Updates the staging configuration for this application.
func (a *App) HasStagingConfig(configRequirement EnvironmentConfigRequirement) error {
	return a.tryUpdateEnvironmentConfig(Staging, configRequirement)
}

// Request cleaning for this application. This marks the application for deletion.
func (a *App) RequestCleanup(requestedBy domain.UserID) {
	if a.cleanupRequested.HasValue() {
		return
	}

	a.apply(AppCleanupRequested{
		ID:               a.id,
		ProductionConfig: a.production,
		StagingConfig:    a.staging,
		Requested:        shared.NewAction(requestedBy),
	})
}

// Delete the application.
func (a *App) Delete(cleanedUp bool) error {
	if !a.cleanupRequested.HasValue() || !cleanedUp {
		return ErrAppCleanupNeeded
	}

	a.apply(AppDeleted{
		ID: a.id,
	})

	return nil
}

func (a *App) ID() AppID                                   { return a.id }
func (a *App) VersionControl() monad.Maybe[VersionControl] { return a.versionControl }
func (a *App) Production() EnvironmentConfig               { return a.production }
func (a *App) Staging() EnvironmentConfig                  { return a.staging }

func (a *App) tryUpdateEnvironmentConfig(
	env Environment,
	updatedConfigRequirement EnvironmentConfigRequirement,
) error {
	if a.cleanupRequested.HasValue() {
		return ErrAppCleanupRequested
	}

	var existingConfig EnvironmentConfig

	switch env {
	case Production:
		existingConfig = a.production
	case Staging:
		existingConfig = a.staging
	default:
		return ErrInvalidEnvironmentName
	}

	updatedConfig, err := updatedConfigRequirement.Met()

	if err != nil {
		return err
	}

	// Same configuration, returns
	if updatedConfig.Equals(existingConfig) {
		return nil
	}

	// Same target, does not update the inner version
	if updatedConfig.target == existingConfig.target {
		updatedConfig.version = existingConfig.version
	}

	a.apply(AppEnvChanged{
		ID:          a.id,
		Environment: env,
		Config:      updatedConfig,
		OldConfig:   existingConfig,
	})

	return nil
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
		switch evt.Environment {
		case Production:
			a.production = evt.Config
		case Staging:
			a.staging = evt.Config
		}
	case AppVersionControlConfigured:
		a.versionControl.Set(evt.Config)
	case AppVersionControlRemoved:
		a.versionControl.Unset()
	case AppCleanupRequested:
		a.cleanupRequested.Set(evt.Requested)
	}

	event.Store(a, e)
}
