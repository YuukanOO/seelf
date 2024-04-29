package provider_test

import (
	"context"
	"testing"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/deployment/infra/provider"
	"github.com/YuukanOO/seelf/pkg/must"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

func Test_Facade(t *testing.T) {
	env := domain.NewEnvironmentConfigRequirement(domain.NewEnvironmentConfig("1"), true, true)
	app := must.Panic(domain.NewApp("app", env, env, "uid"))
	depl := must.Panic(app.NewDeployment(1, dummySourceData{}, domain.Production, "uid"))
	url := domain.NewTargetUrlRequirement(must.Panic(domain.UrlFrom("http://docker.localhost")), true)
	providerConfig := domain.NewProviderConfigRequirement(dummyProviderConfig{}, true)
	target := must.Panic(domain.NewTarget("target", url, providerConfig, "uid"))

	t.Run("should return an error if no provider can handle the payload", func(t *testing.T) {
		sut := provider.NewFacade()

		_, err := sut.Prepare(context.Background(), "payload")

		testutil.ErrorIs(t, domain.ErrNoValidProviderFound, err)
	})

	t.Run("should return an error if no provider can handle the deployment", func(t *testing.T) {
		sut := provider.NewFacade()

		_, err := sut.Deploy(context.Background(), domain.DeploymentContext{}, depl, target)

		testutil.ErrorIs(t, domain.ErrNoValidProviderFound, err)
	})

	t.Run("should return an error if no provider can configure the target", func(t *testing.T) {
		sut := provider.NewFacade()

		_, err := sut.Setup(context.Background(), target)

		testutil.ErrorIs(t, domain.ErrNoValidProviderFound, err)
	})

	t.Run("should return an error if no provider can unconfigure the target", func(t *testing.T) {
		sut := provider.NewFacade()

		err := sut.RemoveConfiguration(context.Background(), target)

		testutil.ErrorIs(t, domain.ErrNoValidProviderFound, err)
	})

	t.Run("should return an error if no provider can cleanup the target", func(t *testing.T) {
		sut := provider.NewFacade()

		err := sut.CleanupTarget(context.Background(), target, domain.CleanupStrategyDefault)

		testutil.ErrorIs(t, domain.ErrNoValidProviderFound, err)
	})

	t.Run("should return an error if no provider can cleanup the app", func(t *testing.T) {
		sut := provider.NewFacade()

		err := sut.Cleanup(context.Background(), app.ID(), target, domain.Production, domain.CleanupStrategyDefault)

		testutil.ErrorIs(t, domain.ErrNoValidProviderFound, err)
	})
}

type (
	dummyProviderConfig struct {
		domain.ProviderConfig
	}

	dummySourceData struct {
		domain.SourceData
	}
)

func (d dummyProviderConfig) Kind() string         { return "dummy" }
func (d dummySourceData) Kind() string             { return "dummy" }
func (d dummySourceData) NeedVersionControl() bool { return false }
