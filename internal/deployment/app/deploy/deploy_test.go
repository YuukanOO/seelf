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
	"github.com/YuukanOO/seelf/internal/deployment/infra/artifact"
	"github.com/YuukanOO/seelf/internal/deployment/infra/memory"
	"github.com/YuukanOO/seelf/internal/deployment/infra/source/raw"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/log"
	"github.com/YuukanOO/seelf/pkg/must"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

func Test_Deploy(t *testing.T) {
	ctx := auth.WithUserID(context.Background(), "some-uid")
	a := must.Panic(domain.NewApp("my-app", domain.NewEnvironmentConfig("1"), domain.NewEnvironmentConfig("1"), "some-uid", domain.AppNamingAvailable))
	logger, _ := log.NewLogger()

	sut := func(
		source domain.Source,
		provider domain.Provider,
		existingDeployments ...*domain.Deployment,
	) bus.RequestHandler[bus.UnitType, deploy.Command] {
		opts := config.Default(config.WithTestDefaults())
		store := memory.NewDeploymentsStore(existingDeployments...)
		artifactManager := artifact.NewLocal(opts, logger)

		t.Cleanup(func() {
			os.RemoveAll(opts.DataDir())
		})

		return deploy.Handler(store, store, artifactManager, source, provider)
	}

	t.Run("should fail if the deployment does not exists", func(t *testing.T) {
		uc := sut(source(nil), provider(nil))
		r, err := uc(ctx, deploy.Command{})

		testutil.ErrorIs(t, apperr.ErrNotFound, err)
		testutil.Equals(t, bus.Unit, r)
	})

	t.Run("should mark the deployment has failed if source does not succeed", func(t *testing.T) {
		srcErr := errors.New("source_failed")
		src := source(srcErr)
		meta, _ := src.Prepare(ctx, a, 42)
		depl, _ := a.NewDeployment(1, meta, domain.Production, "some-uid")
		uc := sut(src, provider(nil), &depl)

		r, err := uc(ctx, deploy.Command{
			AppID:            string(a.ID()),
			DeploymentNumber: 1,
		})

		testutil.ErrorIs(t, srcErr, err)
		testutil.Equals(t, bus.Unit, r)

		evt := testutil.EventIs[domain.DeploymentStateChanged](t, &depl, 2)
		testutil.IsTrue(t, evt.State.StartedAt().HasValue())
		testutil.IsTrue(t, evt.State.FinishedAt().HasValue())
		testutil.Equals(t, srcErr.Error(), evt.State.ErrCode().MustGet())
		testutil.Equals(t, domain.DeploymentStatusFailed, evt.State.Status())
	})

	t.Run("should mark the deployment has failed if provider does not run the deployment successfuly", func(t *testing.T) {
		providerErr := errors.New("run_failed")
		be := provider(providerErr)
		src := source(nil)
		meta, _ := src.Prepare(ctx, a, 42)
		depl, _ := a.NewDeployment(1, meta, domain.Production, "some-uid")
		uc := sut(src, be, &depl)

		r, err := uc(ctx, deploy.Command{
			AppID:            string(a.ID()),
			DeploymentNumber: 1,
		})

		testutil.ErrorIs(t, providerErr, err)
		testutil.Equals(t, bus.Unit, r)
		evt := testutil.EventIs[domain.DeploymentStateChanged](t, &depl, 2)
		testutil.IsTrue(t, evt.State.StartedAt().HasValue())
		testutil.IsTrue(t, evt.State.FinishedAt().HasValue())
		testutil.Equals(t, providerErr.Error(), evt.State.ErrCode().MustGet())
		testutil.Equals(t, domain.DeploymentStatusFailed, evt.State.Status())
	})

	t.Run("should mark the deployment has succeeded if all is good", func(t *testing.T) {
		src := source(nil)
		meta, _ := src.Prepare(ctx, a, 42)
		depl, _ := a.NewDeployment(1, meta, domain.Production, "some-uid")
		uc := sut(src, provider(nil), &depl)

		r, err := uc(ctx, deploy.Command{
			AppID:            string(a.ID()),
			DeploymentNumber: 1,
		})

		testutil.IsNil(t, err)
		testutil.Equals(t, bus.Unit, r)
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

func (*dummySource) Prepare(context.Context, domain.App, any) (domain.SourceData, error) {
	return raw.Data(""), nil
}

func (t *dummySource) Fetch(context.Context, domain.DeploymentContext, domain.Deployment) error {
	return t.err
}

type dummyProvider struct {
	err error
}

func provider(failedWithErr error) domain.Provider {
	return &dummyProvider{failedWithErr}
}

func (b *dummyProvider) Prepare(context.Context, any) (domain.ProviderConfig, error) {
	return nil, nil
}

func (b *dummyProvider) Run(context.Context, domain.DeploymentContext, domain.Deployment) (domain.Services, error) {
	return domain.Services{}, b.err
}

func (b *dummyProvider) Cleanup(context.Context, domain.App) error {
	return nil
}
