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
		source domain.Source,
		backend domain.Backend,
		existingDeployments ...domain.Deployment,
	) (func(context.Context, command.DeployCommand) error, domain.DeploymentsReader) {
		store := memory.NewDeploymentsStore(existingDeployments...)
		return command.Deploy(store, store, source, backend), store
	}

	t.Run("should fail if the deployment does not exists", func(t *testing.T) {
		uc, _ := deploy(source(nil), backend(nil))
		err := uc(ctx, command.DeployCommand{})

		testutil.ErrorIs(t, apperr.ErrNotFound, err)
	})

	t.Run("should mark the deployment has failed if source does not succeed", func(t *testing.T) {
		srcErr := errors.New("source_failed")
		src := source(srcErr)
		meta, _ := src.Prepare(app, 42)
		depl, _ := app.NewDeployment(1, meta, domain.Production, opts, "some-uid")
		uc, reader := deploy(src, backend(nil), depl)

		err := uc(ctx, command.DeployCommand{
			AppID:            string(app.ID()),
			DeploymentNumber: 1,
		})

		depl, _ = reader.GetByID(ctx, domain.DeploymentIDFrom(app.ID(), 1))

		testutil.ErrorIs(t, srcErr, err)

		events := event.Unwrap(&depl)
		evt := events[len(events)-1].(domain.DeploymentStateChanged)

		testutil.IsTrue(t, evt.State.StartedAt().HasValue())
		testutil.IsTrue(t, evt.State.FinishedAt().HasValue())
		testutil.Equals(t, srcErr.Error(), evt.State.ErrCode().MustGet())
		testutil.Equals(t, domain.DeploymentStatusFailed, evt.State.Status())
	})

	t.Run("should mark the deployment has failed if backend does not run the deployment successfuly", func(t *testing.T) {
		backendErr := errors.New("run_failed")
		be := backend(backendErr)
		src := source(nil)
		meta, _ := src.Prepare(app, 42)
		depl, _ := app.NewDeployment(1, meta, domain.Production, opts, "some-uid")
		uc, reader := deploy(src, be, depl)

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
		src := source(nil)
		meta, _ := src.Prepare(app, 42)
		depl, _ := app.NewDeployment(1, meta, domain.Production, opts, "some-uid")
		uc, reader := deploy(src, backend(nil), depl)

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

type dummySource struct {
	err error
}

func source(failedWithErr error) domain.Source {
	return &dummySource{failedWithErr}
}

func (*dummySource) Prepare(domain.App, any) (domain.SourceData, error) {
	return meta{}, nil
}

func (t *dummySource) Fetch(context.Context, domain.Deployment) error {
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

type meta struct{}

func (meta) Discriminator() string { return "test" }
func (m meta) NeedVCS() bool       { return false }
