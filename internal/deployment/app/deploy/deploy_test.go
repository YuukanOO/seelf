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

type initialData struct {
	deployments []*domain.Deployment
	targets     []*domain.Target
}

func Test_Deploy(t *testing.T) {
	ctx := auth.WithUserID(context.Background(), "some-uid")
	logger, _ := log.NewLogger()

	sut := func(
		source domain.Source,
		provider domain.Provider,
		data initialData,
	) bus.RequestHandler[bus.UnitType, deploy.Command] {
		opts := config.Default(config.WithTestDefaults())
		store := memory.NewDeploymentsStore(data.deployments...)
		targetsStore := memory.NewTargetsStore(data.targets...)
		registriesStore := memory.NewRegistriesStore()
		artifactManager := artifact.NewLocal(opts, logger)

		t.Cleanup(func() {
			os.RemoveAll(opts.DataDir())
		})

		return deploy.Handler(store, store, artifactManager, source, provider, targetsStore, registriesStore)
	}

	t.Run("should fail silently if the deployment does not exists", func(t *testing.T) {
		uc := sut(source(nil), provider(nil), initialData{})
		r, err := uc(ctx, deploy.Command{})

		testutil.IsNil(t, err)
		testutil.Equals(t, bus.Unit, r)
	})

	t.Run("should mark the deployment has failed if the target does not exist anymore", func(t *testing.T) {
		app := must.Panic(domain.NewApp("my-app",
			domain.NewEnvironmentConfigRequirement(domain.NewEnvironmentConfig("1"), true, true),
			domain.NewEnvironmentConfigRequirement(domain.NewEnvironmentConfig("1"), true, true), "some-uid"))
		src := source(nil)
		meta := must.Panic(src.Prepare(ctx, app, 42))
		depl := must.Panic(app.NewDeployment(1, meta, domain.Production, "some-uid"))

		uc := sut(src, provider(nil), initialData{
			deployments: []*domain.Deployment{&depl},
		})

		_, err := uc(ctx, deploy.Command{
			AppID:            string(depl.ID().AppID()),
			DeploymentNumber: int(depl.ID().DeploymentNumber()),
		})

		testutil.IsNil(t, err)
		evt := testutil.EventIs[domain.DeploymentStateChanged](t, &depl, 2)
		testutil.Equals(t, apperr.ErrNotFound.Error(), evt.State.ErrCode().MustGet())
	})

	t.Run("should mark the deployment has failed if source does not succeed", func(t *testing.T) {
		target := must.Panic(domain.NewTarget("my-target",
			domain.NewTargetUrlRequirement(must.Panic(domain.UrlFrom("http://localhost")), true),
			domain.NewProviderConfigRequirement(nil, true), "some-uid"))
		target.Configured(target.CurrentVersion(), nil, nil)

		app := must.Panic(domain.NewApp("my-app",
			domain.NewEnvironmentConfigRequirement(domain.NewEnvironmentConfig(target.ID()), true, true),
			domain.NewEnvironmentConfigRequirement(domain.NewEnvironmentConfig(target.ID()), true, true), "some-uid"))
		srcErr := errors.New("source_failed")
		src := source(srcErr)
		meta := must.Panic(src.Prepare(ctx, app, 42))
		depl := must.Panic(app.NewDeployment(1, meta, domain.Production, "some-uid"))
		uc := sut(src, provider(nil), initialData{
			deployments: []*domain.Deployment{&depl},
			targets:     []*domain.Target{&target},
		})

		r, err := uc(ctx, deploy.Command{
			AppID:            string(depl.ID().AppID()),
			DeploymentNumber: int(depl.ID().DeploymentNumber()),
		})

		testutil.IsNil(t, err)
		testutil.Equals(t, bus.Unit, r)

		evt := testutil.EventIs[domain.DeploymentStateChanged](t, &depl, 2)
		testutil.IsTrue(t, evt.State.StartedAt().HasValue())
		testutil.IsTrue(t, evt.State.FinishedAt().HasValue())
		testutil.Equals(t, srcErr.Error(), evt.State.ErrCode().MustGet())
		testutil.Equals(t, domain.DeploymentStatusFailed, evt.State.Status())
	})

	t.Run("should mark the deployment has failed if provider does not run the deployment successfully", func(t *testing.T) {
		target := must.Panic(domain.NewTarget("my-target",
			domain.NewTargetUrlRequirement(must.Panic(domain.UrlFrom("http://localhost")), true),
			domain.NewProviderConfigRequirement(nil, true), "some-uid"))
		target.Configured(target.CurrentVersion(), nil, nil)

		app := must.Panic(domain.NewApp("my-app",
			domain.NewEnvironmentConfigRequirement(domain.NewEnvironmentConfig(target.ID()), true, true),
			domain.NewEnvironmentConfigRequirement(domain.NewEnvironmentConfig(target.ID()), true, true), "some-uid"))
		providerErr := errors.New("run_failed")
		be := provider(providerErr)
		src := source(nil)
		meta := must.Panic(src.Prepare(ctx, app, 42))
		depl := must.Panic(app.NewDeployment(1, meta, domain.Production, "some-uid"))
		uc := sut(src, be, initialData{
			deployments: []*domain.Deployment{&depl},
			targets:     []*domain.Target{&target},
		})

		r, err := uc(ctx, deploy.Command{
			AppID:            string(depl.ID().AppID()),
			DeploymentNumber: int(depl.ID().DeploymentNumber()),
		})

		testutil.IsNil(t, err)
		testutil.Equals(t, bus.Unit, r)
		evt := testutil.EventIs[domain.DeploymentStateChanged](t, &depl, 2)
		testutil.IsTrue(t, evt.State.StartedAt().HasValue())
		testutil.IsTrue(t, evt.State.FinishedAt().HasValue())
		testutil.Equals(t, providerErr.Error(), evt.State.ErrCode().MustGet())
		testutil.Equals(t, domain.DeploymentStatusFailed, evt.State.Status())
	})

	t.Run("should mark the deployment has succeeded if all is good", func(t *testing.T) {
		target := must.Panic(domain.NewTarget("my-target",
			domain.NewTargetUrlRequirement(must.Panic(domain.UrlFrom("http://localhost")), true),
			domain.NewProviderConfigRequirement(nil, true), "some-uid"))
		target.Configured(target.CurrentVersion(), nil, nil)

		app := must.Panic(domain.NewApp("my-app",
			domain.NewEnvironmentConfigRequirement(domain.NewEnvironmentConfig(target.ID()), true, true),
			domain.NewEnvironmentConfigRequirement(domain.NewEnvironmentConfig(target.ID()), true, true), "some-uid"))
		src := source(nil)
		meta := must.Panic(src.Prepare(ctx, app, 42))
		depl := must.Panic(app.NewDeployment(1, meta, domain.Production, "some-uid"))
		uc := sut(src, provider(nil), initialData{
			deployments: []*domain.Deployment{&depl},
			targets:     []*domain.Target{&target},
		})

		r, err := uc(ctx, deploy.Command{
			AppID:            string(depl.ID().AppID()),
			DeploymentNumber: int(depl.ID().DeploymentNumber()),
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
	domain.Provider
	err error
}

func provider(failedWithErr error) domain.Provider {
	return &dummyProvider{
		err: failedWithErr,
	}
}

func (b *dummyProvider) Prepare(context.Context, any, ...domain.ProviderConfig) (domain.ProviderConfig, error) {
	return nil, nil
}

func (b *dummyProvider) Deploy(context.Context, domain.DeploymentContext, domain.Deployment, domain.Target, []domain.Registry) (domain.Services, error) {
	return domain.Services{}, b.err
}
