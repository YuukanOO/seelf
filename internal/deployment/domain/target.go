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
	"github.com/YuukanOO/seelf/pkg/storage"
)

var (
	ErrUrlAlreadyTaken  = apperr.New("url_already_taken")
	ProviderConfigTypes = storage.NewDiscriminatedMapper(func(c ProviderConfig) string { return c.Kind() })
)

type (
	// VALUE OBJECTS

	TargetID              string
	TargetUrlAvailability bool // Represents the availability of an url (ie. is it unique in our system?)

	// ENTITY

	// Represents a target where application could be deployed.
	Target struct {
		event.Emitter

		id       TargetID
		name     string
		url      Url
		provider ProviderConfig
		created  shared.Action[domain.UserID]
	}

	// RELATED SERVICES

	TargetsReader interface {
		GetUrlAvailability(context.Context, Url, ...TargetID) (TargetUrlAvailability, error)
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
		Url      Url
		Provider ProviderConfig
		Created  shared.Action[domain.UserID]
	}
)

func (TargetCreated) Name_() string { return "deployment.event.target_created" }

// Builds a new deployment target.
func NewTarget(
	name string,
	url Url,
	provider ProviderConfig,
	createdBy domain.UserID,
	available TargetUrlAvailability,
) (t Target, err error) {
	if !available {
		return t, ErrUrlAlreadyTaken
	}

	t.apply(TargetCreated{
		ID:       id.New[TargetID](),
		Name:     name,
		Url:      url,
		Provider: provider,
		Created:  shared.NewAction(createdBy),
	})

	return t, nil
}

func TargetFrom(scanner storage.Scanner) (t Target, err error) {
	var (
		createdAt             time.Time
		createdBy             domain.UserID
		providerDiscriminator string
		providerData          string
	)

	err = scanner.Scan(
		&t.id,
		&t.name,
		&t.url,
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

func (t *Target) apply(e event.Event) {
	switch evt := e.(type) {
	case TargetCreated:
		t.id = evt.ID
		t.name = evt.Name
		t.url = evt.Url
		t.provider = evt.Provider
		t.created = evt.Created
	}

	event.Store(t, e)
}
