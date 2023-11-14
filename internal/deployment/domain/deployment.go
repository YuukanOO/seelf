package domain

import (
	"context"
	"time"

	"github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/pkg/apperr"
	shared "github.com/YuukanOO/seelf/pkg/domain"
	"github.com/YuukanOO/seelf/pkg/event"
	"github.com/YuukanOO/seelf/pkg/storage"
)

var (
	ErrCouldNotPromoteProductionDeployment = apperr.New("could_not_promote_production_deployment")
	ErrInvalidSourceDeployment             = apperr.New("invalid_source_deployment")
)

type (
	// VALUE & RELATED OBJECTS

	RunningOrPendingAppDeploymentsCount uint

	// ENTITY

	Deployment struct {
		event.Emitter

		id        DeploymentID
		config    Config
		state     State
		source    SourceData
		requested shared.Action[domain.UserID]
	}

	// RELATED SERVICES

	DeploymentsReader interface {
		GetByID(context.Context, DeploymentID) (Deployment, error)
		GetNextDeploymentNumber(context.Context, AppID) (DeploymentNumber, error)
		GetRunningDeployments(context.Context) ([]Deployment, error)
		GetRunningOrPendingDeploymentsCount(context.Context, AppID) (RunningOrPendingAppDeploymentsCount, error)
	}

	DeploymentsWriter interface {
		Write(context.Context, ...*Deployment) error
	}

	// EVENTS

	DeploymentCreated struct {
		ID        DeploymentID
		Config    Config
		State     State
		Source    SourceData
		Requested shared.Action[domain.UserID]
	}

	DeploymentStateChanged struct {
		ID    DeploymentID
		State State
	}
)

// Creates a new deployment for this app. This method acts as a factory for the deployment
// entity to make sure a new deployment can be created for an app.
func (a App) NewDeployment(
	deployNumber DeploymentNumber,
	meta SourceData,
	env Environment,
	requestedBy domain.UserID,
) (d Deployment, err error) {
	if a.cleanupRequested.HasValue() {
		return d, ErrAppCleanupRequested
	}

	if meta.NeedVCS() && !a.vcs.HasValue() {
		return d, ErrVCSNotConfigured
	}

	d.apply(DeploymentCreated{
		ID:        DeploymentIDFrom(a.id, deployNumber),
		Config:    NewConfig(a, env),
		Source:    meta,
		Requested: shared.NewAction(requestedBy),
	})

	return d, nil
}

// Redeploy the given deployment.
func (a App) Redeploy(
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
func (a App) Promote(
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
		&d.config.appname,
		&d.config.environment,
		&d.config.env,
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

func (d Deployment) ID() DeploymentID                        { return d.id }
func (d Deployment) Config() Config                          { return d.config }
func (d Deployment) Source() SourceData                      { return d.source }
func (d Deployment) Requested() shared.Action[domain.UserID] { return d.requested }

// Mark a deployment has started.
func (d *Deployment) HasStarted() error {
	state, err := d.state.Started()

	if err != nil {
		return err
	}

	d.stateChanged(state)

	return nil
}

// Mark the deployment has ended with availables services or with an error if any.
// The internal status of the deployment will be updated accordingly.
func (d *Deployment) HasEnded(services Services, deploymentErr error) error {
	// No services and no errors, that strange but assume a deployment without services.
	if services == nil && deploymentErr == nil {
		services = Services{}
	}

	var (
		err      error
		newState State
	)

	if deploymentErr != nil {
		newState, err = d.state.Failed(deploymentErr)
	} else {
		newState, err = d.state.Succeeded(services)
	}

	if err != nil {
		return err
	}

	d.stateChanged(newState)

	return nil
}

func (d *Deployment) stateChanged(newState State) {
	d.apply(DeploymentStateChanged{
		ID:    d.id,
		State: newState,
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
