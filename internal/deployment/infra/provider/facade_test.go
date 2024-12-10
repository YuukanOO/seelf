package provider_test

import (
	"context"
	"testing"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/deployment/fixture"
	"github.com/YuukanOO/seelf/internal/deployment/infra/provider"
	"github.com/YuukanOO/seelf/pkg/assert"
)

func Test_Facade(t *testing.T) {
	t.Run("should return an error if no provider can handle the payload", func(t *testing.T) {
		sut := provider.NewFacade()

		_, err := sut.Prepare(context.Background(), "payload")

		assert.ErrorIs(t, domain.ErrNoValidProviderFound, err)
	})

	t.Run("should return an error if no provider can handle the deployment", func(t *testing.T) {
		sut := provider.NewFacade()
		target := fixture.Target()
		depl := fixture.Deployment()

		_, err := sut.Deploy(context.Background(), domain.DeploymentContext{}, depl, target, nil)

		assert.ErrorIs(t, domain.ErrNoValidProviderFound, err)
	})

	t.Run("should return an error if no provider can configure the target", func(t *testing.T) {
		sut := provider.NewFacade()
		target := fixture.Target()

		_, err := sut.Setup(context.Background(), target)

		assert.ErrorIs(t, domain.ErrNoValidProviderFound, err)
	})

	t.Run("should return nil if no provider can unconfigure the target", func(t *testing.T) {
		sut := provider.NewFacade()
		target := fixture.Target()

		err := sut.RemoveConfiguration(context.Background(), target.ID())

		assert.Nil(t, err)
	})

	t.Run("should return an error if no provider can cleanup the target", func(t *testing.T) {
		sut := provider.NewFacade()
		target := fixture.Target()

		err := sut.CleanupTarget(context.Background(), target, domain.CleanupStrategyDefault)

		assert.ErrorIs(t, domain.ErrNoValidProviderFound, err)
	})

	t.Run("should return an error if no provider can cleanup the app", func(t *testing.T) {
		sut := provider.NewFacade()
		app := fixture.App()
		target := fixture.Target()

		err := sut.Cleanup(context.Background(), app.ID(), target, domain.Production, domain.CleanupStrategyDefault)

		assert.ErrorIs(t, domain.ErrNoValidProviderFound, err)
	})
}
