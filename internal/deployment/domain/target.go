package domain

import (
	"context"
	"database/sql/driver"
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
	ErrTargetCleanupNeeded              = apperr.New("target_cleanup_needed")
	ErrTargetCleanupRequested           = apperr.New("target_cleanup_requested")
)

const (
	CleanupStrategyDefault CleanupStrategy = iota // Default strategy, try to remove the target data but returns an error if it fails
	CleanupStrategySkip                           // Skip the cleanup because no resource has been deployed
)

type (
	TargetID                  string
	CleanupStrategy           uint8                                                          // Strategy to use when deleting a target (on the provider side) based on wether it has been successfully configured or not
	TargetEntrypoints         map[AppID]map[Environment]map[EntrypointName]monad.Maybe[Port] // Maps every custom entrypoints managed by this target
	TargetEntrypointsAssigned map[AppID]map[Environment]map[EntrypointName]Port              // Maps every custom entrypoints managed by this target with their assigned port

	// Represents a target where application could be deployed.
	Target struct {
		event.Emitter

		id                TargetID
		name              string
		url               Url
		provider          ProviderConfig
		state             TargetState
		customEntrypoints TargetEntrypoints
		cleanupRequested  monad.Maybe[shared.Action[auth.UserID]]
		created           shared.Action[auth.UserID]
	}

	TargetsReader interface {
		CheckUrlAvailability(context.Context, Url, ...TargetID) (TargetUrlRequirement, error)
		CheckConfigAvailability(context.Context, ProviderConfig, ...TargetID) (ProviderConfigRequirement, error)
		GetByID(context.Context, TargetID) (Target, error)
		GetLocalTarget(context.Context) (Target, error)
	}

	TargetsWriter interface {
		Write(context.Context, ...*Target) error
	}

	TargetCreated struct {
		bus.Notification

		ID          TargetID
		Name        string
		Url         Url
		Provider    ProviderConfig
		State       TargetState
		Entrypoints TargetEntrypoints
		Created     shared.Action[auth.UserID]
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

	TargetEntrypointsChanged struct {
		bus.Notification

		ID          TargetID
		Entrypoints TargetEntrypoints
	}

	TargetCleanupRequested struct {
		bus.Notification

		ID        TargetID
		Requested shared.Action[auth.UserID]
	}

	TargetDeleted struct {
		bus.Notification

		ID TargetID
	}
)

func (TargetCreated) Name_() string            { return "deployment.event.target_created" }
func (TargetStateChanged) Name_() string       { return "deployment.event.target_state_changed" }
func (TargetRenamed) Name_() string            { return "deployment.event.target_renamed" }
func (TargetUrlChanged) Name_() string         { return "deployment.event.target_url_changed" }
func (TargetProviderChanged) Name_() string    { return "deployment.event.target_provider_changed" }
func (TargetEntrypointsChanged) Name_() string { return "deployment.event.target_entrypoints_changed" }
func (TargetCleanupRequested) Name_() string   { return "deployment.event.target_cleanup_requested" }
func (TargetDeleted) Name_() string            { return "deployment.event.target_deleted" }

func (e TargetStateChanged) WentToConfiguringState() bool {
	return e.State.status == TargetStatusConfiguring
}

// Builds a new deployment target.
func NewTarget(
	name string,
	urlRequirement TargetUrlRequirement,
	providerRequirement ProviderConfigRequirement,
	createdBy auth.UserID,
) (t Target, err error) {
	url, err := urlRequirement.Met()

	if err != nil {
		return t, err
	}

	provider, err := providerRequirement.Met()

	if err != nil {
		return t, err
	}

	t.apply(TargetCreated{
		ID:          id.New[TargetID](),
		Name:        name,
		Url:         url.Root(),
		Provider:    provider,
		State:       newTargetState(),
		Entrypoints: make(TargetEntrypoints),
		Created:     shared.NewAction(createdBy),
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
		&t.customEntrypoints,
		&deleteRequestedAt,
		&deleteRequestedBy,
		&createdAt,
		&createdBy,
	)

	if err != nil {
		return t, err
	}

	if requestedAt, isSet := deleteRequestedAt.TryGet(); isSet {
		t.cleanupRequested.Set(
			shared.ActionFrom(auth.UserID(deleteRequestedBy.MustGet()), requestedAt),
		)
	}

	t.provider, err = ProviderConfigTypes.From(providerDiscriminator, providerData)
	t.created = shared.ActionFrom(createdBy, createdAt)

	return t, err
}

// Rename the target.
func (t *Target) Rename(name string) error {
	if t.cleanupRequested.HasValue() {
		return ErrTargetCleanupRequested
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
func (t *Target) HasUrl(urlRequirement TargetUrlRequirement) error {
	if t.cleanupRequested.HasValue() {
		return ErrTargetCleanupRequested
	}

	url, err := urlRequirement.Met()

	if err != nil {
		return err
	}

	if t.url.Equals(url) {
		return nil
	}

	t.apply(TargetUrlChanged{
		ID:  t.id,
		Url: url.Root(),
	})

	t.reconfigure()

	return nil
}

// Update the target provider information.
func (t *Target) HasProvider(providerRequirement ProviderConfigRequirement) error {
	if t.cleanupRequested.HasValue() {
		return ErrTargetCleanupRequested
	}

	provider, err := providerRequirement.Met()

	if err != nil {
		return err
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

	t.apply(TargetProviderChanged{
		ID:       t.id,
		Provider: provider,
	})

	t.reconfigure()

	return nil
}

// Check the target availability and returns an appropriate error.
func (t *Target) CheckAvailability() error {
	if t.state.status == TargetStatusConfiguring {
		return ErrTargetConfigurationInProgress
	}

	if t.cleanupRequested.HasValue() {
		return ErrTargetCleanupRequested
	}

	if t.state.status != TargetStatusReady {
		return ErrTargetConfigurationFailed
	}

	return nil
}

// Force the target reconfiguration.
func (t *Target) Reconfigure() error {
	if t.cleanupRequested.HasValue() {
		return ErrTargetCleanupRequested
	}

	if t.state.status == TargetStatusConfiguring {
		return ErrTargetConfigurationInProgress
	}

	t.reconfigure()

	return nil
}

// Mark the target (in the given version) has configured (by an external system).
// If the given version does not match the current one, nothing will be done.
func (t *Target) Configured(version time.Time, assigned TargetEntrypointsAssigned, err error) {
	if !t.state.Configured(version, err) {
		return
	}

	if err == nil && t.customEntrypoints.assign(assigned) {
		t.apply(TargetEntrypointsChanged{
			ID:          t.id,
			Entrypoints: t.customEntrypoints,
		})
	}

	t.apply(TargetStateChanged{
		ID:    t.id,
		State: t.state,
	})
}

// Inform the target that it should exposes entrypoints inside the services array
// for the given application environment.
// Only custom entrypoints will be added to the target.
// If needed (new or removed entrypoints), a configuration will be triggered.
func (t *Target) ExposeEntrypoints(app AppID, env Environment, services Services) {
	// Target is being deleted, no need to reconfigure anything
	if t.cleanupRequested.HasValue() || services == nil {
		return
	}

	if !t.customEntrypoints.merge(app, env, services.CustomEntrypoints()) {
		return
	}

	t.raiseEntrypointsChangedAndReconfigure()
}

// Un-expose entrypoints for the given application and environments. If no environment is given,
// all entrypoints for the application will be removed.
// If the entrypoints have changed, a configuration will be triggered.
func (t *Target) UnExposeEntrypoints(app AppID, envs ...Environment) {
	if t.cleanupRequested.HasValue() {
		return
	}

	if !t.customEntrypoints.remove(app, envs...) {
		return
	}

	t.raiseEntrypointsChangedAndReconfigure()
}

// Request the target cleanup, meaning it will be deleted with all its related data.
func (t *Target) RequestCleanup(apps HasAppsOnTarget, by auth.UserID) error {
	if t.cleanupRequested.HasValue() {
		return nil
	}

	if apps {
		return ErrTargetInUse
	}

	if t.state.status == TargetStatusConfiguring {
		return ErrTargetConfigurationInProgress
	}

	t.apply(TargetCleanupRequested{
		ID:        t.id,
		Requested: shared.NewAction(by),
	})

	return nil
}

// Check the target cleanup strategy to determine how the target resources should be handled.
func (t *Target) CleanupStrategy(deployments HasRunningOrPendingDeploymentsOnTarget) (CleanupStrategy, error) {
	if deployments {
		return CleanupStrategyDefault, ErrRunningOrPendingDeployments
	}

	switch t.state.status {
	case TargetStatusConfiguring:
		return CleanupStrategyDefault, ErrTargetConfigurationInProgress
	case TargetStatusReady:
		return CleanupStrategyDefault, nil
	default:
		// Never reachable or target has been marked for deletion, no way to update it anymore, just skip the cleanup
		if !t.state.lastReadyVersion.HasValue() || t.cleanupRequested.HasValue() {
			return CleanupStrategySkip, nil
		}

		return CleanupStrategyDefault, ErrTargetConfigurationFailed
	}
}

// Check the cleanup strategy for a specific application to determine how related resources
// should be handled.
func (t *Target) AppCleanupStrategy(
	ongoing HasRunningOrPendingDeploymentsOnAppTargetEnv,
	successful HasSuccessfulDeploymentsOnAppTargetEnv,
) (CleanupStrategy, error) {
	// Target will be deleted, skip the cleanup right away
	if t.cleanupRequested.HasValue() {
		return CleanupStrategySkip, nil
	}

	// Still running deployments on the target for the app, do not allow the cleanup yet
	if ongoing {
		return CleanupStrategyDefault, ErrRunningOrPendingDeployments
	}

	// No successful deployments, skip the cleanup
	if !successful {
		return CleanupStrategySkip, nil
	}

	switch t.state.status {
	case TargetStatusConfiguring:
		return CleanupStrategyDefault, ErrTargetConfigurationInProgress
	case TargetStatusReady:
		return CleanupStrategyDefault, nil
	default:
		return CleanupStrategyDefault, ErrTargetConfigurationFailed
	}
}

// Deletes the target.
func (t *Target) Delete(cleanedUp bool) error {
	if !t.cleanupRequested.HasValue() || !cleanedUp {
		return ErrTargetCleanupNeeded
	}

	t.apply(TargetDeleted{
		ID: t.id,
	})

	return nil
}

func (t *Target) ID() TargetID                         { return t.id }
func (t *Target) Url() Url                             { return t.url }
func (t *Target) Provider() ProviderConfig             { return t.provider }
func (t *Target) CustomEntrypoints() TargetEntrypoints { return t.customEntrypoints } // FIXME: Should we return a copy?
func (t *Target) CurrentVersion() time.Time            { return t.state.version }

// Returns true if the given configuration version is different from the current one.
func (t *Target) IsOutdated(version time.Time) bool {
	return t.state.IsOutdated(version)
}

func (t *Target) reconfigure() {
	t.state.Reconfigure()

	t.apply(TargetStateChanged{
		ID:    t.id,
		State: t.state,
	})
}

func (t *Target) raiseEntrypointsChangedAndReconfigure() {
	t.apply(TargetEntrypointsChanged{
		ID:          t.id,
		Entrypoints: t.customEntrypoints,
	})

	t.reconfigure()
}

func (t *Target) apply(e event.Event) {
	switch evt := e.(type) {
	case TargetCreated:
		t.id = evt.ID
		t.name = evt.Name
		t.url = evt.Url
		t.provider = evt.Provider
		t.state = evt.State
		t.created = evt.Created
		t.customEntrypoints = evt.Entrypoints
	case TargetRenamed:
		t.name = evt.Name
	case TargetUrlChanged:
		t.url = evt.Url
	case TargetProviderChanged:
		t.provider = evt.Provider
	case TargetEntrypointsChanged:
		t.customEntrypoints = evt.Entrypoints
	case TargetCleanupRequested:
		t.cleanupRequested.Set(evt.Requested)
	case TargetStateChanged:
		t.state = evt.State
	}

	event.Store(t, e)
}

func (e TargetEntrypoints) Value() (driver.Value, error) { return storage.ValueJSON(e) }
func (e *TargetEntrypoints) Scan(value any) error        { return storage.ScanJSON(value, e) }

func (e TargetEntrypoints) merge(app AppID, env Environment, entrypoints []Entrypoint) (updated bool) {
	appEntries, found := e[app]

	if !found {
		appEntries = make(map[Environment]map[EntrypointName]monad.Maybe[Port])
		e[app] = appEntries
	}

	envEntries, found := appEntries[env]

	if !found {
		envEntries = make(map[EntrypointName]monad.Maybe[Port])
		appEntries[env] = envEntries
	}

	// Remove old entries
	for existing := range envEntries {
		stillExist := false

		for _, entry := range entrypoints {
			if entry.name == existing {
				stillExist = true
				break
			}
		}

		if stillExist {
			continue
		}

		updated = true
		delete(envEntries, existing)
	}

	// Add new entries but do not overwrite existing ones
	for _, entrypoint := range entrypoints {
		if _, found := envEntries[entrypoint.name]; found {
			continue
		}

		updated = true
		envEntries[entrypoint.name] = monad.None[Port]()
	}

	// Clean useless entries
	if len(envEntries) == 0 {
		delete(appEntries, env)
	}

	if len(appEntries) == 0 {
		delete(e, app)
	}

	return
}

func (e TargetEntrypoints) remove(app AppID, envs ...Environment) (updated bool) {
	appEntries, found := e[app]

	if !found {
		return
	}

	if len(envs) == 0 {
		delete(e, app)
		return true
	}

	for _, env := range envs {
		_, found := appEntries[env]

		if !found {
			continue
		}

		delete(appEntries, env)
		updated = true
	}

	if len(appEntries) == 0 {
		delete(e, app)
	}

	return
}

func (e TargetEntrypoints) assign(mapping TargetEntrypointsAssigned) (updated bool) {
	for app, envEntries := range mapping {
		for env, entries := range envEntries {
			for name, assignedPort := range entries {
				port, found := e[app][env][name]

				if !found {
					continue
				}

				port.Set(assignedPort)
				e[app][env][name] = port
				updated = true
			}
		}
	}

	return
}

// Sets the entrypoint port for the given entrypoint.
// It will create the needed structure as needed.
func (e TargetEntrypointsAssigned) Set(app AppID, env Environment, name EntrypointName, port Port) {
	// Updates the assigned map to keep track of new ports assigned to this target
	appEntries, found := e[app]

	if !found {
		appEntries = make(map[Environment]map[EntrypointName]Port)
		e[app] = appEntries
	}

	envEntries, found := appEntries[env]

	if !found {
		envEntries = make(map[EntrypointName]Port)
		appEntries[env] = envEntries
	}

	envEntries[name] = port
}
