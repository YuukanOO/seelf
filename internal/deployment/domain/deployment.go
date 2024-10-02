package domain

import (
	"context"
	"time"

	"github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/bus"
	shared "github.com/YuukanOO/seelf/pkg/domain"
	"github.com/YuukanOO/seelf/pkg/event"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/storage"
)

var (
	ErrCouldNotPromoteProductionDeployment = apperr.New("could_not_promote_production_deployment")
	ErrRunningOrPendingDeployments         = apperr.New("running_or_pending_deployments")
	ErrInvalidSourceDeployment             = apperr.New("invalid_source_deployment")
)

type (
	HasRunningOrPendingDeploymentsOnTarget       bool
	HasRunningOrPendingDeploymentsOnAppTargetEnv bool
	HasSuccessfulDeploymentsOnAppTargetEnv       bool

	Deployment struct {
		event.Emitter

		id        DeploymentID
		config    ConfigSnapshot
		state     DeploymentState
		source    SourceData
		requested shared.Action[domain.UserID]
	}

	DeploymentsReader interface {
		GetByID(context.Context, DeploymentID) (Deployment, error)
		GetLastDeployment(context.Context, AppID, Environment) (Deployment, error)
		GetNextDeploymentNumber(context.Context, AppID) (DeploymentNumber, error)
		HasRunningOrPendingDeploymentsOnTarget(context.Context, TargetID) (HasRunningOrPendingDeploymentsOnTarget, error)
		// Retrieve running or pending deployments count for a specific app, target and environment and the successful deployments count
		// during the specified interval.
		HasDeploymentsOnAppTargetEnv(context.Context, AppID, TargetID, Environment, shared.TimeInterval) (HasRunningOrPendingDeploymentsOnAppTargetEnv, HasSuccessfulDeploymentsOnAppTargetEnv, error)
	}

	FailCriteria struct {
		Status      monad.Maybe[DeploymentStatus]
		Target      monad.Maybe[TargetID]
		App         monad.Maybe[AppID]
		Environment monad.Maybe[Environment]
	}

	DeploymentsWriter interface {
		FailDeployments(context.Context, error, FailCriteria) error // Fail all deployments matching the given filters
		Write(context.Context, ...*Deployment) error
	}

	DeploymentCreated struct {
		bus.Notification

		ID        DeploymentID
		Config    ConfigSnapshot
		State     DeploymentState
		Source    SourceData
		Requested shared.Action[domain.UserID]
	}

	DeploymentStateChanged struct {
		bus.Notification

		ID     DeploymentID
		Config ConfigSnapshot
		State  DeploymentState
	}
)

func (DeploymentCreated) Name_() string      { return "deployment.event.deployment_created" }
func (DeploymentStateChanged) Name_() string { return "deployment.event.deployment_state_changed" }

func (e DeploymentStateChanged) HasSucceeded() bool {
	return e.State.status == DeploymentStatusSucceeded
}

// Creates a new deployment for this app. This method acts as a factory for the deployment
// entity to make sure a new deployment can be created for an app.
func (a *App) NewDeployment(
	deployNumber DeploymentNumber,
	meta SourceData,
	env Environment,
	requestedBy domain.UserID,
) (d Deployment, err error) {
	if a.cleanupRequested.HasValue() {
		return d, ErrAppCleanupRequested
	}

	if meta.NeedVersionControl() && !a.versionControl.HasValue() {
		return d, ErrVersionControlNotConfigured
	}

	conf, err := a.configSnapshotFor(env)

	if err != nil {
		return d, err
	}

	d.apply(DeploymentCreated{
		ID:        DeploymentIDFrom(a.id, deployNumber),
		Config:    conf,
		Source:    meta,
		Requested: shared.NewAction(requestedBy),
	})

	return d, nil
}

func DeploymentFrom(scanner storage.Scanner) (d Deployment, err error) {
	var (
		requestedAt             time.Time
		requestedBy             domain.UserID
		sourceMetaDiscriminator string
		sourceMetaData          string
	)

	err = scanner.Scan(
		&d.id.appID,
		&d.id.deploymentNumber,
		&d.config.appid,
		&d.config.appname,
		&d.config.environment,
		&d.config.target,
		&d.config.vars,
		&d.state.status,
		&d.state.errcode,
		&d.state.services,
		&d.state.startedAt,
		&d.state.finishedAt,
		&sourceMetaDiscriminator,
		&sourceMetaData,
		&requestedAt,
		&requestedBy,
	)

	if err != nil {
		return d, err
	}

	d.source, err = SourceDataTypes.From(sourceMetaDiscriminator, sourceMetaData)
	d.requested = shared.ActionFrom(requestedBy, requestedAt)

	return d, err
}

// Redeploy the given deployment.
func (a *App) Redeploy(
	source Deployment,
	deployNumber DeploymentNumber,
	requestedBy domain.UserID,
) (d Deployment, err error) {
	if source.id.appID != a.id {
		return d, ErrInvalidSourceDeployment
	}

	return a.NewDeployment(deployNumber, source.source, source.config.environment, requestedBy)
}

// Promote the given deployment to the production environment
func (a *App) Promote(
	source Deployment,
	deployNumber DeploymentNumber,
	requestedBy domain.UserID,
) (d Deployment, err error) {
	if source.config.environment.IsProduction() {
		return d, ErrCouldNotPromoteProductionDeployment
	}

	if source.id.appID != a.id {
		return d, ErrInvalidSourceDeployment
	}

	return a.NewDeployment(deployNumber, source.source, Production, requestedBy)
}

func (d *Deployment) ID() DeploymentID                        { return d.id }
func (d *Deployment) Config() ConfigSnapshot                  { return d.config }
func (d *Deployment) Source() SourceData                      { return d.source }
func (d *Deployment) Requested() shared.Action[domain.UserID] { return d.requested }

// Mark a deployment has started.
func (d *Deployment) HasStarted() error {
	if err := d.state.started(); err != nil {
		return err
	}

	d.stateChanged()

	return nil
}

// Mark the deployment has ended with available services or with an error if any.
// The internal status of the deployment will be updated accordingly.
func (d *Deployment) HasEnded(services Services, deploymentErr error) error {
	if err := d.state.ended(services, deploymentErr); err != nil {
		return err
	}

	d.stateChanged()

	return nil
}

func (d *Deployment) stateChanged() {
	d.apply(DeploymentStateChanged{
		ID:     d.id,
		Config: d.config,
		State:  d.state,
	})
}

func (d *Deployment) apply(e event.Event) {
	switch evt := e.(type) {
	case DeploymentCreated:
		d.id = evt.ID
		d.config = evt.Config
		d.state = evt.State
		d.source = evt.Source
		d.requested = evt.Requested
	case DeploymentStateChanged:
		d.state = evt.State
	}

	event.Store(d, e)
}

var (
	ErrNotInPendingState = apperr.New("not_in_pending_state")
	ErrNotInRunningState = apperr.New("not_in_running_state")
)

const (
	DeploymentStatusPending DeploymentStatus = iota
	DeploymentStatusRunning
	DeploymentStatusFailed
	DeploymentStatusSucceeded
)

type (
	DeploymentStatus uint8

	// Holds together information related to the current deployment state. With a value
	// object, it is easier to validate consistency between all those related properties.
	// The default value represents a pending state.
	DeploymentState struct {
		status     DeploymentStatus
		errcode    monad.Maybe[string]
		services   monad.Maybe[Services]
		startedAt  monad.Maybe[time.Time]
		finishedAt monad.Maybe[time.Time]
	}
)

func (s *DeploymentState) started() error {
	if s.status != DeploymentStatusPending {
		return ErrNotInPendingState
	}

	s.status = DeploymentStatusRunning
	s.startedAt.Set(time.Now().UTC())

	return nil
}

func (s *DeploymentState) ended(services Services, err error) error {
	if s.status != DeploymentStatusRunning {
		return ErrNotInRunningState
	}

	s.finishedAt.Set(time.Now().UTC())

	if err != nil {
		s.status = DeploymentStatusFailed
		s.errcode.Set(err.Error())
		return nil
	}

	s.status = DeploymentStatusSucceeded

	if services == nil {
		services = make(Services, 0)
	}

	s.services.Set(services)

	return nil
}

func (s DeploymentState) Status() DeploymentStatus           { return s.status }
func (s DeploymentState) ErrCode() monad.Maybe[string]       { return s.errcode }
func (s DeploymentState) Services() monad.Maybe[Services]    { return s.services }
func (s DeploymentState) StartedAt() monad.Maybe[time.Time]  { return s.startedAt }
func (s DeploymentState) FinishedAt() monad.Maybe[time.Time] { return s.finishedAt }
