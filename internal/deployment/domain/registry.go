package domain

import (
	"context"
	"time"

	auth "github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/pkg/bus"
	shared "github.com/YuukanOO/seelf/pkg/domain"
	"github.com/YuukanOO/seelf/pkg/event"
	"github.com/YuukanOO/seelf/pkg/id"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/storage"
)

type (
	RegistryID string

	// Represents a custom registry to pull images from, not particularly tied
	// to Docker.
	Registry struct {
		event.Emitter

		id          RegistryID
		name        string
		url         Url
		credentials monad.Maybe[Credentials]
		created     shared.Action[auth.UserID]
	}

	RegistriesReader interface {
		CheckUrlAvailability(context.Context, Url, ...RegistryID) (RegistryUrlRequirement, error)
		GetByID(context.Context, RegistryID) (Registry, error)
		GetAll(context.Context) ([]Registry, error)
	}

	RegistriesWriter interface {
		Write(context.Context, ...*Registry) error
	}

	RegistryCreated struct {
		bus.Notification

		ID      RegistryID
		Name    string
		Url     Url
		Created shared.Action[auth.UserID]
	}

	RegistryRenamed struct {
		bus.Notification

		ID   RegistryID
		Name string
	}

	RegistryUrlChanged struct {
		bus.Notification

		ID  RegistryID
		Url Url
	}

	RegistryCredentialsChanged struct {
		bus.Notification

		ID          RegistryID
		Credentials Credentials
	}

	RegistryCredentialsRemoved struct {
		bus.Notification

		ID RegistryID
	}

	RegistryDeleted struct {
		bus.Notification

		ID RegistryID
	}
)

func (RegistryCreated) Name_() string    { return "deployment.event.registry_created" }
func (RegistryRenamed) Name_() string    { return "deployment.event.registry_renamed" }
func (RegistryUrlChanged) Name_() string { return "deployment.event.registry_url_changed" }
func (RegistryDeleted) Name_() string    { return "deployment.event.registry_deleted" }

func (RegistryCredentialsChanged) Name_() string {
	return "deployment.event.registry_credentials_changed"
}
func (RegistryCredentialsRemoved) Name_() string {
	return "deployment.event.registry_credentials_removed"
}

// Declare a new custom registry at the given URL.
func NewRegistry(
	name string,
	urlRequirement RegistryUrlRequirement,
	uid auth.UserID,
) (r Registry, err error) {
	url, err := urlRequirement.Met()

	if err != nil {
		return r, err
	}

	r.apply(RegistryCreated{
		ID:      id.New[RegistryID](),
		Name:    name,
		Url:     url,
		Created: shared.NewAction(uid),
	})

	return r, err
}

// Recreates a registry from the persistent storage.
func RegistryFrom(scanner storage.Scanner) (r Registry, err error) {
	var (
		version   event.Version
		username  monad.Maybe[string]
		password  monad.Maybe[string]
		createdAt time.Time
		createdBy auth.UserID
	)

	err = scanner.Scan(
		&r.id,
		&r.name,
		&r.url,
		&username,
		&password,
		&createdAt,
		&createdBy,
		&version,
	)

	if err != nil {
		return r, err
	}

	event.Hydrate(&r, version)

	r.created = shared.ActionFrom(createdBy, createdAt)

	if usr, isSet := username.TryGet(); isSet {
		r.credentials.Set(NewCredentials(usr, password.Get("")))
	}

	return r, err
}

// Renames the registry.
func (r *Registry) Rename(name string) {
	if r.name == name {
		return
	}

	r.apply(RegistryRenamed{
		ID:   r.id,
		Name: name,
	})
}

// Updates the registry URL.
func (r *Registry) HasUrl(urlRequirement RegistryUrlRequirement) error {
	url, err := urlRequirement.Met()

	if err != nil {
		return err
	}

	if url == r.url {
		return nil
	}

	r.apply(RegistryUrlChanged{
		ID:  r.id,
		Url: url,
	})

	return nil
}

// Set the authentication configuration for the registry.
func (r *Registry) UseAuthentication(credentials Credentials) {
	if existing, isSet := r.credentials.TryGet(); isSet && existing == credentials {
		return
	}

	r.apply(RegistryCredentialsChanged{
		ID:          r.id,
		Credentials: credentials,
	})
}

// Remove the authentication configuration for the registry.
func (r *Registry) RemoveAuthentication() {
	if !r.credentials.HasValue() {
		return
	}

	r.apply(RegistryCredentialsRemoved{
		ID: r.id,
	})
}

func (r *Registry) Delete() {
	r.apply(RegistryDeleted{
		ID: r.id,
	})
}

func (r *Registry) ID() RegistryID                        { return r.id }
func (r *Registry) Name() string                          { return r.name }
func (r *Registry) Url() Url                              { return r.url }
func (r *Registry) Credentials() monad.Maybe[Credentials] { return r.credentials }

func (r *Registry) apply(e event.Event) {
	switch v := e.(type) {
	case RegistryCreated:
		r.id = v.ID
		r.url = v.Url
		r.name = v.Name
		r.created = v.Created
	case RegistryRenamed:
		r.name = v.Name
	case RegistryCredentialsChanged:
		r.credentials.Set(v.Credentials)
	case RegistryCredentialsRemoved:
		r.credentials.Unset()
	case RegistryUrlChanged:
		r.url = v.Url
	}

	event.Store(r, e)
}
