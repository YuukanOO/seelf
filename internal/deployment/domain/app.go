package domain

import (
	"context"
	"time"

	"github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/bus"
	shared "github.com/YuukanOO/seelf/pkg/domain"
	"github.com/YuukanOO/seelf/pkg/event"
	"github.com/YuukanOO/seelf/pkg/flag"
	"github.com/YuukanOO/seelf/pkg/id"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/storage"
)

var (
	ErrInvalidAppNaming                  = apperr.New("invalid_app_naming")
	ErrVCSNotConfigured                  = apperr.New("vcs_not_configured")
	ErrAppCleanupNeeded                  = apperr.New("app_cleanup_needed")
	ErrAppCleanupRequested               = apperr.New("app_cleanup_requested")
	ErrAppHasRunningOrPendingDeployments = apperr.New("app_has_running_or_pending_deployments")
)

const (
	AppNamingProductionTargetNotFound AppNamingAvailability = 1 << iota
	AppNamingStagingTargetNotFound
	AppNamingTakenInProduction
	AppNamingTakenInStaging
	AppNamingAvailable
)

const (
	TargetAppNamingTargetNotFound TargetAppNamingAvailability = 1 << iota
	TargetAppNamingTaken
	TargetAppNamingAvailable
)

type (
	// VALUE OBJECTS
	AppID string

	// Represents a naming availability. This one is represented with flags because
	// there can be many reasons for a name to be unavailable and I want to represents
	// all of them so the application layer could be clearer with the user.
	AppNamingAvailability uint8

	TargetAppNamingAvailability uint8 // Same as the AppNamingAvailability but for a specific target environment

	// ENTITY

	App struct {
		event.Emitter

		id               AppID
		name             AppName
		vcs              monad.Maybe[VCSConfig]
		production       EnvironmentConfig
		staging          EnvironmentConfig
		cleanupRequested monad.Maybe[shared.Action[domain.UserID]]
		created          shared.Action[domain.UserID]
	}

	// RELATED SERVICES

	AppsReader interface {
		GetAppNamingAvailability(context.Context, AppName, TargetID, TargetID) (AppNamingAvailability, error)
		GetTargetAppNamingAvailability(context.Context, AppID, Environment, TargetID) (TargetAppNamingAvailability, error)
		GetByID(context.Context, AppID) (App, error)
	}

	AppsWriter interface {
		Write(context.Context, ...*App) error
	}

	// EVENTS

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
	}

	AppVCSConfigured struct {
		bus.Notification

		ID     AppID
		Config VCSConfig
	}

	AppVCSRemoved struct {
		bus.Notification

		ID AppID
	}

	AppCleanupRequested struct {
		bus.Notification

		ID        AppID
		Requested shared.Action[domain.UserID]
	}

	AppDeleted struct {
		bus.Notification

		ID AppID
	}
)

func (AppCreated) Name_() string          { return "deployment.event.app_created" }
func (AppEnvChanged) Name_() string       { return "deployment.event.app_env_changed" }
func (AppVCSConfigured) Name_() string    { return "deployment.event.app_vcs_configured" }
func (AppVCSRemoved) Name_() string       { return "deployment.event.app_vcs_removed" }
func (AppCleanupRequested) Name_() string { return "deployment.event.app_cleanup_requested" }
func (AppDeleted) Name_() string          { return "deployment.event.app_deleted" }

// Instantiates a new App.
func NewApp(
	name AppName,
	production EnvironmentConfig,
	staging EnvironmentConfig,
	available AppNamingAvailability,
	createdBy domain.UserID,
) (app App, err error) {
	if available != AppNamingAvailable {
		return app, ErrInvalidAppNaming
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
		&a.production.vars,
		&a.staging.target,
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
		vcs := NewVCSConfig(u)

		if tok, isSet := token.TryGet(); isSet {
			vcs.Authenticated(tok)
		}

		a.vcs.Set(vcs)
	}

	return a, err
}

// Sets an app version control configuration.
func (a *App) UseVersionControl(config VCSConfig) {
	if existing, isSet := a.vcs.TryGet(); isSet && config.Equals(existing) {
		return
	}

	a.apply(AppVCSConfigured{
		ID:     a.id,
		Config: config,
	})
}

// Removes the version control configuration from the app.
func (a *App) RemoveVersionControl() {
	if !a.vcs.HasValue() {
		return
	}

	a.apply(AppVCSRemoved{
		ID: a.id,
	})
}

// Updates the production configuration for this application.
func (a *App) WithProductionConfig(config EnvironmentConfig, available TargetAppNamingAvailability) error {
	return a.tryUpdateEnvironmentConfig(Production, config, available)
}

// Updates the staging configuration for this application.
func (a *App) WithStagingConfig(config EnvironmentConfig, available TargetAppNamingAvailability) error {
	return a.tryUpdateEnvironmentConfig(Staging, config, available)
}

// Request cleaning for this application. This marks the application for deletion.
func (a *App) RequestCleanup(requestedBy domain.UserID) {
	if a.cleanupRequested.HasValue() {
		return
	}

	a.apply(AppCleanupRequested{
		ID:        a.id,
		Requested: shared.NewAction(requestedBy),
	})
}

// Delete the application. This will only succeed if there are no running or pending deployments and
// a cleanup request has been made.
func (a *App) Delete(deployments RunningOrPendingAppDeploymentsCount) error {
	if !a.cleanupRequested.HasValue() {
		return ErrAppCleanupNeeded
	}

	if deployments > 0 {
		return ErrAppHasRunningOrPendingDeployments
	}

	a.apply(AppDeleted{
		ID: a.id,
	})

	return nil
}

func (a *App) ID() AppID                   { return a.id }
func (a *App) VCS() monad.Maybe[VCSConfig] { return a.vcs }

func (a *App) tryUpdateEnvironmentConfig(
	env Environment,
	updatedConfig EnvironmentConfig,
	available TargetAppNamingAvailability,
) error {
	var existingConfig EnvironmentConfig

	switch env {
	case Production:
		existingConfig = a.production
	case Staging:
		existingConfig = a.staging
	default:
		return ErrInvalidEnvironmentName
	}

	// Same configuration, returns
	if updatedConfig.Equals(existingConfig) {
		return nil
	}

	// Target different, let's check naming uniqueness
	if existingConfig.target != updatedConfig.target &&
		available != TargetAppNamingAvailable {
		return ErrInvalidAppNaming
	}

	a.apply(AppEnvChanged{
		ID:          a.id,
		Environment: env,
		Config:      updatedConfig,
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
	case AppVCSConfigured:
		a.vcs.Set(evt.Config)
	case AppVCSRemoved:
		a.vcs.Unset()
	case AppCleanupRequested:
		a.cleanupRequested.Set(evt.Requested)
	}

	event.Store(a, e)
}

// Converts the AppNamingAvailability to a more detailed error.
func (a AppNamingAvailability) Error(env Environment) error {
	switch env {
	case Production:
		if flag.IsSet(a, AppNamingProductionTargetNotFound) {
			return apperr.ErrNotFound
		}

		if flag.IsSet(a, AppNamingTakenInProduction) {
			return ErrInvalidAppNaming
		}
	case Staging:
		if flag.IsSet(a, AppNamingStagingTargetNotFound) {
			return apperr.ErrNotFound
		}

		if flag.IsSet(a, AppNamingTakenInStaging) {
			return ErrInvalidAppNaming
		}
	}

	return nil
}

// Converts the TargetAppNamingAvailability to a more detailed error.
func (a TargetAppNamingAvailability) Error() error {
	if flag.IsSet(a, TargetAppNamingTargetNotFound) {
		return apperr.ErrNotFound
	}

	if flag.IsSet(a, TargetAppNamingTaken) {
		return ErrInvalidAppNaming
	}

	return nil
}
