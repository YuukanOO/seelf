package command_test

import (
	"context"
	"errors"
	"testing"

	auth "github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/internal/deployment/app/command"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/deployment/infra/memory"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/event"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

func Test_Deploy(t *testing.T) {
	opts := options{}
	ctx := auth.WithUserID(context.Background(), "some-uid")
	app := domain.NewApp("my-app", "some-uid")

	deploy := func(
		trigger domain.Trigger,
		backend domain.Backend,
		existingDeployments ...domain.Deployment,
	) (func(context.Context, command.DeployCommand) error, domain.DeploymentsReader) {
		store := memory.NewDeploymentsStore(existingDeployments...)
		return command.Deploy(store, store, trigger, backend), store
	}

	t.Run("should fail if the deployment does not exists", func(t *testing.T) {
		uc, _ := deploy(trigger(nil), backend(nil))
		err := uc(ctx, command.DeployCommand{})

		testutil.ErrorIs(t, apperr.ErrNotFound, err)
	})

	t.Run("should mark the deployment has failed if trigger does not succeed", func(t *testing.T) {
		triggerErr := errors.New("trigger_failed")
		tr := trigger(triggerErr)
		meta, _ := tr.Prepare(app, 42)
		depl, _ := app.NewDeployment(1, meta, domain.Production, opts, "some-uid")
		uc, reader := deploy(tr, backend(nil), depl)

		err := uc(ctx, command.DeployCommand{
			AppID:            string(app.ID()),
			DeploymentNumber: 1,
		})

		depl, _ = reader.GetByID(ctx, domain.DeploymentIDFrom(app.ID(), 1))

		testutil.ErrorIs(t, triggerErr, err)

		events := event.Unwrap(&depl)
		evt := events[len(events)-1].(domain.DeploymentStateChanged)

		testutil.IsTrue(t, evt.State.StartedAt().HasValue())
		testutil.IsTrue(t, evt.State.FinishedAt().HasValue())
		testutil.Equals(t, triggerErr.Error(), evt.State.ErrCode().MustGet())
		testutil.Equals(t, domain.DeploymentStatusFailed, evt.State.Status())
	})

	t.Run("should mark the deployment has failed if backend does not run the deployment successfuly", func(t *testing.T) {
		backendErr := errors.New("run_failed")
		be := backend(backendErr)
		tr := trigger(nil)
		meta, _ := tr.Prepare(app, 42)
		depl, _ := app.NewDeployment(1, meta, domain.Production, opts, "some-uid")
		uc, reader := deploy(tr, be, depl)

		err := uc(ctx, command.DeployCommand{
			AppID:            string(app.ID()),
			DeploymentNumber: 1,
		})
		depl, _ = reader.GetByID(ctx, domain.DeploymentIDFrom(app.ID(), 1))

		testutil.ErrorIs(t, backendErr, err)

		events := event.Unwrap(&depl)
		evt := events[len(events)-1].(domain.DeploymentStateChanged)

		testutil.IsTrue(t, evt.State.StartedAt().HasValue())
		testutil.IsTrue(t, evt.State.FinishedAt().HasValue())
		testutil.Equals(t, domain.DeploymentStatusFailed, evt.State.Status())
	})

	t.Run("should mark the deployment has succeeded if all is good", func(t *testing.T) {
		tr := trigger(nil)
		meta, _ := tr.Prepare(app, 42)
		depl, _ := app.NewDeployment(1, meta, domain.Production, opts, "some-uid")
		uc, reader := deploy(tr, backend(nil), depl)

		err := uc(ctx, command.DeployCommand{
			AppID:            string(app.ID()),
			DeploymentNumber: 1,
		})
		depl, _ = reader.GetByID(ctx, domain.DeploymentIDFrom(app.ID(), 1))

		testutil.IsNil(t, err)

		events := event.Unwrap(&depl)
		evt := events[len(events)-1].(domain.DeploymentStateChanged)

		testutil.IsTrue(t, evt.State.StartedAt().HasValue())
		testutil.IsTrue(t, evt.State.FinishedAt().HasValue())
		testutil.Equals(t, domain.DeploymentStatusSucceeded, evt.State.Status())
	})
}

const kind domain.Kind = "dummy"

type dummyTrigger struct {
	err error
}

func trigger(failedWithErr error) domain.Trigger {
	return &dummyTrigger{failedWithErr}
}

func (*dummyTrigger) Prepare(domain.App, any) (domain.Meta, error) {
	return domain.NewMeta(kind, ""), nil
}

func (t *dummyTrigger) Fetch(context.Context, domain.Deployment) error {
	return t.err
}

type dummyBackend struct {
	err error
}

func backend(failedWithErr error) domain.Backend {
	return &dummyBackend{failedWithErr}
}

func (b *dummyBackend) Run(context.Context, domain.Deployment) (domain.Services, error) {
	return domain.Services{}, b.err
}

func (b *dummyBackend) Cleanup(context.Context, domain.App) error {
	return nil
}
