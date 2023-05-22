package domain

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/YuukanOO/seelf/internal/auth/domain"
	shared "github.com/YuukanOO/seelf/internal/shared/domain"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/event"
	"github.com/YuukanOO/seelf/pkg/storage"
)

var ErrCouldNotPromoteProductionDeployment = apperr.New("could_not_promote_production_deployment")

type (
	// VALUE & RELATED OBJECTS

	RunningOrPendingAppDeploymentsCount uint

	// Template data used to build a deployment path.
	DeploymentTemplateData struct {
		Number      DeploymentNumber
		Environment Environment
	}

	// Since the build directory is a template, materialize it with this tiny interface.
	// Making it a template enables the user to configure how the deployment directory
	// should be built.
	DeploymentDirTemplate interface {
		Execute(DeploymentTemplateData) string
	}

	// ENTITY

	Deployment struct {
		event.Emitter

		id        DeploymentID
		path      string
		config    Config
		state     State
		trigger   Meta
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
		Path      string
		Config    Config
		State     State
		Trigger   Meta
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
	meta Meta,
	env Environment,
	tmpl DeploymentDirTemplate,
	requestedBy domain.UserID,
) (d Deployment, err error) {
	if a.cleanupRequested.HasValue() {
		return d, ErrAppCleanupRequested
	}

	if meta.kind.IsVCS() && !a.vcs.HasValue() {
		return d, ErrVCSNotConfigured
	}

	now := time.Now().UTC()
	// Use the deployment requested time as a prefix for the deployment logfile
	logfilename := fmt.Sprintf("%d-%s-%d.deployment.log", now.Unix(), a.id, deployNumber)
	// Build the final deployment path, this is where sources will be put under
	path := filepath.Join(a.Path(), tmpl.Execute(DeploymentTemplateData{deployNumber, env}))

	d.apply(DeploymentCreated{
		ID:        DeploymentIDFrom(a.id, deployNumber),
		Path:      path,
		Config:    NewConfig(a, env),
		State:     NewState(logfilename),
		Trigger:   meta,
		Requested: shared.ActionFrom(requestedBy, now),
	})

	return d, nil
}

// Redeploy the given deployment.
func (a App) Redeploy(
	source Deployment,
	deployNumber DeploymentNumber,
	tmpl DeploymentDirTemplate,
	requestedBy domain.UserID,
) (d Deployment, err error) {
	return a.NewDeployment(deployNumber, source.trigger, source.config.environment, tmpl, requestedBy)
}

// Promote the given deployment to the production environment
func (a App) Promote(
	source Deployment,
	deployNumber DeploymentNumber,
	tmpl DeploymentDirTemplate,
	requestedBy domain.UserID,
) (d Deployment, err error) {
	if source.config.environment.IsProduction() {
		return d, ErrCouldNotPromoteProductionDeployment
	}

	return a.NewDeployment(deployNumber, source.trigger, Production, tmpl, requestedBy)
}

func DeploymentFrom(scanner storage.Scanner) (d Deployment, err error) {
	var (
		requestedAt time.Time
		requestedBy domain.UserID
	)

	err = scanner.Scan(
		&d.id.appID,
		&d.id.deploymentNumber,
		&d.path,
		&d.config.appname,
		&d.config.environment,
		&d.config.env,
		&d.state.status,
		&d.state.logfile,
		&d.state.errcode,
		&d.state.services,
		&d.state.startedAt,
		&d.state.finishedAt,
		&d.trigger.kind,
		&d.trigger.data,
		&requestedAt,
		&requestedBy,
	)

	d.requested = shared.ActionFrom(requestedBy, requestedAt)

	return d, err
}

func (d Deployment) ID() DeploymentID { return d.id }
func (d Deployment) Config() Config   { return d.config }
func (d Deployment) Trigger() Meta    { return d.trigger }

// Retrieve the deployment path relative to the given directories.
func (d Deployment) Path(relativeTo ...string) string {
	relativeTo = append(relativeTo, d.path)
	return filepath.Join(relativeTo...)
}

// Retrieve the path where the log for this deployment is stored relative to the
// given directories.
func (d Deployment) LogPath(relativeTo ...string) string {
	relativeTo = append(relativeTo, d.state.LogFile())
	return filepath.Join(relativeTo...)
}

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
		d.path = evt.Path
		d.config = evt.Config
		d.state = evt.State
		d.trigger = evt.Trigger
		d.requested = evt.Requested
	case DeploymentStateChanged:
		d.state = evt.State
	}

	event.Store(d, e)
}
