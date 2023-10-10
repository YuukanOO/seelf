package domain

import (
	"context"
	"path/filepath"
	"time"

	"github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/pkg/apperr"
	shared "github.com/YuukanOO/seelf/pkg/domain"
	"github.com/YuukanOO/seelf/pkg/event"
	"github.com/YuukanOO/seelf/pkg/id"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/storage"
)

var (
	ErrAppNameAlreadyTaken               = apperr.New("app_name_already_taken")
	ErrVCSNotConfigured                  = apperr.New("vcs_not_configured")
	ErrAppCleanupNeeded                  = apperr.New("app_cleanup_needed")
	ErrAppCleanupRequested               = apperr.New("app_cleanup_requested")
	ErrAppHasRunningOrPendingDeployments = apperr.New("app_has_running_or_pending_deployments")
)

type (
	// VALUE OBJECTS

	AppID         string
	UniqueAppName AppName // Represents the unique name of an app and will be used as a subdomain.

	// ENTITY

	App struct {
		event.Emitter

		id               AppID
		name             UniqueAppName
		vcs              monad.Maybe[VCSConfig]
		env              monad.Maybe[EnvironmentsEnv]
		cleanupRequested monad.Maybe[shared.Action[domain.UserID]]
		created          shared.Action[domain.UserID]
	}

	// RELATED SERVICES

	AppsReader interface {
		IsNameUnique(context.Context, AppName) (UniqueAppName, error)
		GetByID(context.Context, AppID) (App, error)
	}

	AppsWriter interface {
		Write(context.Context, ...*App) error
	}

	// EVENTS

	AppCreated struct {
		ID      AppID
		Name    UniqueAppName
		Created shared.Action[domain.UserID]
	}

	AppEnvChanged struct {
		ID  AppID
		Env EnvironmentsEnv
	}

	AppEnvRemoved struct {
		ID AppID
	}

	AppVCSConfigured struct {
		ID     AppID
		Config VCSConfig
	}

	AppVCSRemoved struct {
		ID AppID
	}

	AppCleanupRequested struct {
		ID        AppID
		Requested shared.Action[domain.UserID]
	}

	AppDeleted struct {
		ID AppID
	}
)

// Instantiates a new App.
func NewApp(name UniqueAppName, createdBy domain.UserID) (app App) {
	app.apply(AppCreated{
		ID:      id.New[AppID](),
		Name:    name,
		Created: shared.NewAction(createdBy),
	})
	return app
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
		&a.env,
		&cleanupRequestedAt,
		&cleanupRequestedBy,
		&createdAt,
		&createdBy,
	)

	a.created = shared.ActionFrom(createdBy, createdAt)

	if cleanupRequestedAt.HasValue() {
		a.cleanupRequested = a.cleanupRequested.WithValue(
			shared.ActionFrom(domain.UserID(cleanupRequestedBy.MustGet()), cleanupRequestedAt.MustGet()),
		)
	}

	// vcs url has been set, reconstitute the vcs config
	if url.HasValue() {
		vcs := NewVCSConfig(url.MustGet())

		if token.HasValue() {
			vcs = vcs.Authenticated(token.MustGet())
		}

		a.vcs = a.vcs.WithValue(vcs)
	}

	return a, err
}

// Sets an app version control configuration.
func (a *App) UseVersionControl(config VCSConfig) {
	if a.vcs.HasValue() && config.Equals(a.vcs.MustGet()) {
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

// Store environement variables per env and per services for this application.
func (a *App) HasEnvironmentVariables(vars EnvironmentsEnv) {
	if a.env.HasValue() && vars.Equals(a.env.MustGet()) {
		return
	}

	a.apply(AppEnvChanged{
		ID:  a.id,
		Env: vars,
	})
}

// Removes all environment variables for this application.
func (a *App) RemoveEnvironmentVariables() {
	if !a.env.HasValue() {
		return
	}

	a.apply(AppEnvRemoved{
		ID: a.id,
	})
}

// Request backend cleaning for this application. This marks the application for deletion.
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

func (a App) ID() AppID                   { return a.id }
func (a App) VCS() monad.Maybe[VCSConfig] { return a.vcs }

// Retrieve the application directory relative to the given path.
func (a App) Path(relativeTo ...string) string {
	relativeTo = append(relativeTo, string(a.id))
	return filepath.Join(relativeTo...)
}

// Retrieve environments variables per service for the given deployment environment
func (a App) envFor(e Environment) (m monad.Maybe[ServicesEnv]) {
	if !a.env.HasValue() {
		return m
	}

	vars, exists := a.env.MustGet()[e]

	if !exists {
		return m
	}

	return m.WithValue(vars)
}

func (a *App) apply(e event.Event) {
	switch evt := e.(type) {
	case AppCreated:
		a.id = evt.ID
		a.name = evt.Name
		a.created = evt.Created
	case AppEnvChanged:
		a.env = a.env.WithValue(evt.Env)
	case AppEnvRemoved:
		a.env = a.env.None()
	case AppVCSConfigured:
		a.vcs = a.vcs.WithValue(evt.Config)
	case AppVCSRemoved:
		a.vcs = a.vcs.None()
	case AppCleanupRequested:
		a.cleanupRequested = a.cleanupRequested.WithValue(evt.Requested)
	}

	event.Store(a, e)
}
