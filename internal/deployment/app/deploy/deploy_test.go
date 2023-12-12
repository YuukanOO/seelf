package deploy_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/YuukanOO/seelf/cmd/config"
	auth "github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/internal/deployment/app/deploy"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/deployment/infra"
	"github.com/YuukanOO/seelf/internal/deployment/infra/memory"
	"github.com/YuukanOO/seelf/internal/deployment/infra/source/raw"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/log"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

func Test_Deploy(t *testing.T) {
	ctx := auth.WithUserID(context.Background(), "some-uid")
	a := domain.NewApp("my-app", "some-uid")
	logger := log.NewLogger(false)

	sut := func(
		source domain.Source,
		backend domain.Backend,
		existingDeployments ...*domain.Deployment,
	) bus.RequestHandler[bool, deploy.Command] {
		opts := config.Default(config.WithTestDefaults())
		store := memory.NewDeploymentsStore(existingDeployments...)
		artifactManager := infra.NewLocalArtifactManager(opts, logger)

		t.Cleanup(func() {
			os.RemoveAll(opts.DataDir())
		})

		return deploy.Handler(store, store, artifactManager, source, backend)
	}

	t.Run("should fail if the deployment does not exists", func(t *testing.T) {
		uc := sut(source(nil), backend(nil))
		success, err := uc(ctx, deploy.Command{})

		testutil.ErrorIs(t, apperr.ErrNotFound, err)
		testutil.IsFalse(t, success)
	})

	t.Run("should mark the deployment has failed if source does not succeed", func(t *testing.T) {
		srcErr := errors.New("source_failed")
		src := source(srcErr)
		meta, _ := src.Prepare(a, 42)
		depl, _ := a.NewDeployment(1, meta, domain.Production, "some-uid")
		uc := sut(src, backend(nil), &depl)

		success, err := uc(ctx, deploy.Command{
			AppID:            string(a.ID()),
			DeploymentNumber: 1,
		})

		testutil.ErrorIs(t, srcErr, err)
		testutil.IsFalse(t, success)

		evt := testutil.EventIs[domain.DeploymentStateChanged](t, &depl, 2)
		testutil.IsTrue(t, evt.State.StartedAt().HasValue())
		testutil.IsTrue(t, evt.State.FinishedAt().HasValue())
		testutil.Equals(t, srcErr.Error(), evt.State.ErrCode().MustGet())
		testutil.Equals(t, domain.DeploymentStatusFailed, evt.State.Status())
	})

	t.Run("should mark the deployment has failed if backend does not run the deployment successfuly", func(t *testing.T) {
		backendErr := errors.New("run_failed")
		be := backend(backendErr)
		src := source(nil)
		meta, _ := src.Prepare(a, 42)
		depl, _ := a.NewDeployment(1, meta, domain.Production, "some-uid")
		uc := sut(src, be, &depl)

		success, err := uc(ctx, deploy.Command{
			AppID:            string(a.ID()),
			DeploymentNumber: 1,
		})

		testutil.ErrorIs(t, backendErr, err)
		testutil.IsFalse(t, success)
		evt := testutil.EventIs[domain.DeploymentStateChanged](t, &depl, 2)
		testutil.IsTrue(t, evt.State.StartedAt().HasValue())
		testutil.IsTrue(t, evt.State.FinishedAt().HasValue())
		testutil.Equals(t, backendErr.Error(), evt.State.ErrCode().MustGet())
		testutil.Equals(t, domain.DeploymentStatusFailed, evt.State.Status())
	})

	t.Run("should mark the deployment has succeeded if all is good", func(t *testing.T) {
		src := source(nil)
		meta, _ := src.Prepare(a, 42)
		depl, _ := a.NewDeployment(1, meta, domain.Production, "some-uid")
		uc := sut(src, backend(nil), &depl)

		success, err := uc(ctx, deploy.Command{
			AppID:            string(a.ID()),
			DeploymentNumber: 1,
		})

		testutil.IsNil(t, err)
		testutil.IsTrue(t, success)
		evt := testutil.EventIs[domain.DeploymentStateChanged](t, &depl, 2)
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
	return raw.Data(""), nil
}

func (t *dummySource) Fetch(context.Context, string, domain.DeploymentLogger, domain.Deployment) error {
	return t.err
}

type dummyBackend struct {
	err error
}

func backend(failedWithErr error) domain.Backend {
	return &dummyBackend{failedWithErr}
}

func (b *dummyBackend) Run(context.Context, string, domain.DeploymentLogger, domain.Deployment) (domain.Services, error) {
	return domain.Services{}, b.err
}

func (b *dummyBackend) Cleanup(context.Context, domain.App) error {
	return nil
}
