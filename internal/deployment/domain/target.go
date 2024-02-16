package domain

import (
	"context"
	"time"

	auth "github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/bus"
	shared "github.com/YuukanOO/seelf/pkg/domain"
	"github.com/YuukanOO/seelf/pkg/event"
	"github.com/YuukanOO/seelf/pkg/id"
	"github.com/YuukanOO/seelf/pkg/storage"
)

var (
	ErrDomainAlreadyTaken = apperr.New("domain_already_taken")
	ErrConfigAlreadyTaken = apperr.New("config_already_taken")
)

type (
	// VALUE OBJECTS

	TargetID                 string
	TargetDomainAvailability bool // Represents the availability of a domain (ie. is it unique in our system?)
	TargetConfigAvailability bool // Represents the availability of a target configuration

	// ENTITY

	// Represents a target where application could be deployed.
	Target struct {
		event.Emitter

		id       TargetID
		name     string
		domain   Url
		provider ProviderConfig
		created  shared.Action[auth.UserID]
	}

	// RELATED SERVICES

	TargetsReader interface {
		GetDomainAvailability(context.Context, Url, ...TargetID) (TargetDomainAvailability, error)
		GetConfigAvailability(context.Context, ProviderConfig, ...TargetID) (TargetConfigAvailability, error)
		GetByID(context.Context, TargetID) (Target, error)
	}

	TargetsWriter interface {
		Write(context.Context, ...*Target) error
	}

	// EVENTS

	TargetCreated struct {
		bus.Notification

		ID       TargetID
		Name     string
		Domain   Url
		Provider ProviderConfig
		Created  shared.Action[auth.UserID]
	}
)

func (TargetCreated) Name_() string { return "deployment.event.target_created" }

// Builds a new deployment target.
func NewTarget(
	name string,
	domain Url,
	available TargetDomainAvailability,
	provider ProviderConfig,
	configAvailable TargetConfigAvailability,
	createdBy auth.UserID,
) (t Target, err error) {
	if !available {
		return t, ErrDomainAlreadyTaken
	}

	if !configAvailable {
		return t, ErrConfigAlreadyTaken
	}

	t.apply(TargetCreated{
		ID:       id.New[TargetID](),
		Name:     name,
		Domain:   domain,
		Provider: provider,
		Created:  shared.NewAction(createdBy),
	})

	return t, nil
}

func TargetFrom(scanner storage.Scanner) (t Target, err error) {
	var (
		createdAt             time.Time
		createdBy             auth.UserID
		providerDiscriminator string
		providerData          string
	)

	err = scanner.Scan(
		&t.id,
		&t.name,
		&t.domain,
		&providerDiscriminator,
		&providerData,
		&createdAt,
		&createdBy,
	)

	if err != nil {
		return t, err
	}

	t.provider, err = ProviderConfigTypes.From(providerDiscriminator, providerData)
	t.created = shared.ActionFrom(createdBy, createdAt)

	return t, err
}

func (t Target) ID() TargetID             { return t.id }
func (t Target) Provider() ProviderConfig { return t.provider }

func (t *Target) apply(e event.Event) {
	switch evt := e.(type) {
	case TargetCreated:
		t.id = evt.ID
		t.name = evt.Name
		t.domain = evt.Domain
		t.provider = evt.Provider
		t.created = evt.Created
	}

	event.Store(t, e)
}

func (a TargetConfigAvailability) Error() error {
	if !a {
		return ErrConfigAlreadyTaken
	}

	return nil
}

func (a TargetDomainAvailability) Error() error {
	if !a {
		return ErrDomainAlreadyTaken
	}

	return nil
}
