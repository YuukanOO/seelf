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
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/storage"
)

var (
	ErrUrlAlreadyTaken                  = apperr.New("url_already_taken")
	ErrConfigAlreadyTaken               = apperr.New("config_already_taken")
	ErrTargetInUse                      = apperr.New("target_in_use")
	ErrTargetConfigurationInProgress    = apperr.New("target_configuration_in_progress")
	ErrTargetConfigurationFailed        = apperr.New("target_configuration_failed")
	ErrTargetProviderUpdateNotPermitted = apperr.New("target_provider_update_not_permitted")
	ErrTargetDeleteRequestNeeded        = apperr.New("target_delete_request_needed")
	ErrTargetDeleteRequested            = apperr.New("target_delete_requested")
)

const (
	TargetCleanupStrategyDefault TargetCleanupStrategy = iota // Default strategy, try to remove the target data but returns an error if it fails
	TargetCleanupStrategyForce                                // Force the cleanup
	TargetCleanupStrategySkip                                 // Skip the cleanup because the target was never configured properly
)

type (
	// VALUE OBJECTS

	TargetID                 string
	TargetUrlAvailability    bool  // Represents the availability of a domain (ie. is it unique in our system?)
	TargetConfigAvailability bool  // Represents the availability of a target configuration
	TargetCleanupStrategy    uint8 // Strategy to use when deleting a target (on the provider side) based on wether it has been successfully configured or not

	// ENTITY

	// Represents a target where application could be deployed.
	Target struct {
		event.Emitter

		id              TargetID
		name            string
		url             Url
		provider        ProviderConfig
		state           TargetState
		deleteRequested monad.Maybe[shared.Action[auth.UserID]]
		created         shared.Action[auth.UserID]
	}

	// RELATED SERVICES

	TargetsReader interface {
		GetUrlAvailability(context.Context, Url, ...TargetID) (TargetUrlAvailability, error)
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
		Url      Url
		Provider ProviderConfig
		State    TargetState
		Created  shared.Action[auth.UserID]
	}

	TargetStateChanged struct {
		bus.Notification

		ID    TargetID
		State TargetState
	}

	TargetRenamed struct {
		bus.Notification

		ID   TargetID
		Name string
	}

	TargetUrlChanged struct {
		bus.Notification

		ID  TargetID
		Url Url
	}

	TargetProviderChanged struct {
		bus.Notification

		ID       TargetID
		Provider ProviderConfig
	}

	TargetDeleteRequested struct {
		bus.Notification

		ID        TargetID
		Requested shared.Action[auth.UserID]
	}

	TargetDeleted struct {
		bus.Notification

		ID TargetID
	}
)

func (TargetCreated) Name_() string         { return "deployment.event.target_created" }
func (TargetStateChanged) Name_() string    { return "deployment.event.target_state_changed" }
func (TargetRenamed) Name_() string         { return "deployment.event.target_renamed" }
func (TargetUrlChanged) Name_() string      { return "deployment.event.target_url_changed" }
func (TargetProviderChanged) Name_() string { return "deployment.event.target_provider_changed" }
func (TargetDeleteRequested) Name_() string { return "deployment.event.target_delete_requested" }
func (TargetDeleted) Name_() string         { return "deployment.event.target_deleted" }

// Builds a new deployment target.
func NewTarget(
	name string,
	domain Url,
	available TargetUrlAvailability,
	provider ProviderConfig,
	configAvailable TargetConfigAvailability,
	createdBy auth.UserID,
) (t Target, err error) {
	if !available {
		return t, ErrUrlAlreadyTaken
	}

	if !configAvailable {
		return t, ErrConfigAlreadyTaken
	}

	t.apply(TargetCreated{
		ID:       id.New[TargetID](),
		Name:     name,
		Url:      domain,
		Provider: provider,
		State:    newTargetState(),
		Created:  shared.NewAction(createdBy),
	})

	return t, nil
}

func TargetFrom(scanner storage.Scanner) (t Target, err error) {
	var (
		createdAt             time.Time
		createdBy             auth.UserID
		deleteRequestedAt     monad.Maybe[time.Time]
		deleteRequestedBy     monad.Maybe[string]
		providerDiscriminator string
		providerData          string
	)

	err = scanner.Scan(
		&t.id,
		&t.name,
		&t.url,
		&providerDiscriminator,
		&providerData,
		&t.state.status,
		&t.state.version,
		&t.state.errcode,
		&t.state.lastReadyVersion,
		&deleteRequestedAt,
		&deleteRequestedBy,
		&createdAt,
		&createdBy,
	)

	if err != nil {
		return t, err
	}

	if requestedAt, isSet := deleteRequestedAt.TryGet(); isSet {
		t.deleteRequested.Set(
			shared.ActionFrom(auth.UserID(deleteRequestedBy.MustGet()), requestedAt),
		)
	}

	t.provider, err = ProviderConfigTypes.From(providerDiscriminator, providerData)
	t.created = shared.ActionFrom(createdBy, createdAt)

	return t, err
}

// Rename the target.
func (t *Target) Rename(name string) error {
	if t.deleteRequested.HasValue() {
		return ErrTargetDeleteRequested
	}

	if name == t.name {
		return nil
	}

	t.apply(TargetRenamed{
		ID:   t.id,
		Name: name,
	})

	return nil
}

// Update the internal domain used by this target.
func (t *Target) HasUrl(url Url, availability TargetUrlAvailability) error {
	if t.deleteRequested.HasValue() {
		return ErrTargetDeleteRequested
	}

	if t.url.Equals(url) {
		return nil
	}

	if !availability {
		return ErrUrlAlreadyTaken
	}

	t.apply(TargetUrlChanged{
		ID:  t.id,
		Url: url,
	})

	t.Reconfigure()

	return nil
}

// Update the target provider information.
func (t *Target) HasProvider(provider ProviderConfig, availability TargetConfigAvailability) error {
	if t.deleteRequested.HasValue() {
		return ErrTargetDeleteRequested
	}

	// Kind or fingerprint changed, it means the target host has probably changed,
	// for now, just forbid it (the user will have to delete/create a new target),
	// because what does it means for the outdated one? Should we remove the configuration?
	if provider.Kind() != t.provider.Kind() ||
		provider.Fingerprint() != t.provider.Fingerprint() {
		return ErrTargetProviderUpdateNotPermitted
	}

	if t.provider.Equals(provider) {
		return nil
	}

	if !availability {
		return ErrConfigAlreadyTaken
	}

	t.apply(TargetProviderChanged{
		ID:       t.id,
		Provider: provider,
	})

	t.Reconfigure()

	return nil
}

// Check the target availability and returns an appropriate error.
// The boolean returned indicates if the target has been ready at least once
// in the past.
func (t *Target) CheckAvailability() (beenReadyAtLeastOnce bool, err error) {
	beenReadyAtLeastOnce = t.state.lastReadyVersion.HasValue()

	if t.state.status == TargetStatusConfiguring {
		err = ErrTargetConfigurationInProgress
		return
	}

	if t.deleteRequested.HasValue() {
		err = ErrTargetDeleteRequested
		return
	}

	if t.state.status != TargetStatusReady {
		err = ErrTargetConfigurationFailed
		return
	}

	return
}

// Force the target reconfiguration.
func (t *Target) Reconfigure() {
	t.state.Reconfigure()

	t.apply(TargetStateChanged{
		ID:    t.id,
		State: t.state,
	})
}

// Mark the target (in the given version) has configured (by an external system).
// If the given version does not match the current one, nothing will be done.
func (t *Target) Configured(version time.Time, err error) {
	if !t.state.Configured(version, err) {
		return
	}

	t.apply(TargetStateChanged{
		ID:    t.id,
		State: t.state,
	})
}

// Request the target deletion, meaning it will be deleted with all its related data.
func (t *Target) RequestDelete(apps AppsOnTargetCount, by auth.UserID) error {
	if t.deleteRequested.HasValue() {
		return nil
	}

	if apps > 0 {
		return ErrTargetInUse
	}

	t.apply(TargetDeleteRequested{
		ID:        t.id,
		Requested: shared.NewAction(by),
	})

	return nil
}

// Deletes the target. It will fails if there are at least one deployment using it currently or if the
// target status does not allow the deletion at that moment.
func (t *Target) Delete(deployments RunningDeploymentsOnTargetCount) (TargetCleanupStrategy, error) {

	if !t.deleteRequested.HasValue() {
		return TargetCleanupStrategyDefault, ErrTargetDeleteRequestNeeded
	}

	if deployments > 0 {
		return TargetCleanupStrategyDefault, ErrTargetInUse
	}

	if t.state.status == TargetStatusConfiguring {
		return TargetCleanupStrategyDefault, ErrTargetConfigurationInProgress
	}

	t.apply(TargetDeleted{
		ID: t.id,
	})

	var strategy TargetCleanupStrategy

	// If the target configuration has failed, 2 solutions
	if t.state.status != TargetStatusReady {
		if t.state.lastReadyVersion.HasValue() {
			// The target was ready in the past, force the cleanup in the provider side
			// because it means the user wants to remove a target which has seelf stuff on it
			// but we cannot reach it anymore.
			strategy = TargetCleanupStrategyForce
		} else {
			// Else, the target was never reachable, we can safely skip the cleanup.
			strategy = TargetCleanupStrategySkip
		}
	}

	return strategy, nil
}

func (t *Target) ID() TargetID             { return t.id }
func (t *Target) Url() Url                 { return t.url }
func (t *Target) Provider() ProviderConfig { return t.provider }

// Returns true if the given configuration version is different from the current one.
func (t *Target) IsOutdated(version time.Time) bool {
	return t.state.IsOutdated(version)
}

func (t *Target) apply(e event.Event) {
	var isStateChanged bool

	switch evt := e.(type) {
	case TargetCreated:
		t.id = evt.ID
		t.name = evt.Name
		t.url = evt.Url
		t.provider = evt.Provider
		t.state = evt.State
		t.created = evt.Created
	case TargetRenamed:
		t.name = evt.Name
	case TargetUrlChanged:
		t.url = evt.Url
	case TargetProviderChanged:
		t.provider = evt.Provider
	case TargetDeleteRequested:
		t.deleteRequested.Set(evt.Requested)
	case TargetStateChanged:
		t.state = evt.State
		isStateChanged = true
	}

	// Prevent multiple TargetStateChanged events to be dispatched in the same transaction
	// because it would be useless to configure the target multiple times.
	if isStateChanged {
		event.Replace(t, e)
		return
	}

	event.Store(t, e)
}

func (a TargetConfigAvailability) Error() error {
	if !a {
		return ErrConfigAlreadyTaken
	}

	return nil
}

func (a TargetUrlAvailability) Error() error {
	if !a {
		return ErrUrlAlreadyTaken
	}

	return nil
}
