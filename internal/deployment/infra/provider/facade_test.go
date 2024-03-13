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
	app := must.Panic(domain.NewApp("app", domain.NewEnvironmentConfig("1"), domain.NewEnvironmentConfig("1"),
		domain.AppNamingProductionAvailable|domain.AppNamingStagingAvailable, "uid"))
	depl := must.Panic(app.NewDeployment(1, dummySourceData{}, domain.Production, "uid"))
	target := must.Panic(domain.NewTarget("target", must.Panic(domain.UrlFrom("http://docker.localhost")), true, dummyProviderConfig{}, true, "uid"))

	t.Run("should return an error if no provider can handle the payload", func(t *testing.T) {
		sut := provider.NewFacade()

		_, err := sut.Prepare(context.Background(), "payload")

		testutil.ErrorIs(t, domain.ErrNoValidProviderFound, err)
	})

	t.Run("should return an error if no provider can handle the deployment", func(t *testing.T) {
		sut := provider.NewFacade()

		_, err := sut.Run(context.Background(), domain.DeploymentContext{}, depl, target)

		testutil.ErrorIs(t, domain.ErrNoValidProviderFound, err)
	})

	t.Run("should do nothing if no provider mark the target as stale", func(t *testing.T) {
		sut := provider.NewFacade()

		testutil.IsNil(t, sut.Stale(context.Background(), "target"))
	})

	t.Run("should return an error if no provider can cleanup the target", func(t *testing.T) {
		sut := provider.NewFacade()

		err := sut.CleanupTarget(context.Background(), target)

		testutil.ErrorIs(t, domain.ErrNoValidProviderFound, err)
	})

	t.Run("should return an error if no provider can cleanup the app", func(t *testing.T) {
		sut := provider.NewFacade()

		err := sut.Cleanup(context.Background(), app.ID(), target, domain.Production)

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

func (d dummyProviderConfig) Kind() string { return "dummy" }
func (d dummySourceData) Kind() string     { return "dummy" }
func (d dummySourceData) NeedVCS() bool    { return false }
