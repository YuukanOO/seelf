package promote_test

import (
	"context"
	"testing"

	authfixture "github.com/YuukanOO/seelf/internal/auth/fixture"
	"github.com/YuukanOO/seelf/internal/deployment/app/promote"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/deployment/fixture"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/assert"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/bus/spy"
	shared "github.com/YuukanOO/seelf/pkg/domain"
)

func Test_Promote(t *testing.T) {

	arrange := func(tb testing.TB, seed ...fixture.SeedBuilder) (
		bus.RequestHandler[int, promote.Command],
		context.Context,
		spy.Dispatcher,
	) {
		context := fixture.PrepareDatabase(tb, seed...)
		return promote.Handler(context.AppsStore, context.DeploymentsStore, context.DeploymentsStore), context.Context, context.Dispatcher
	}

	t.Run("should fail if application does not exist", func(t *testing.T) {
		handler, ctx, _ := arrange(t)

		num, err := handler(ctx, promote.Command{
			AppID: "some-app-id",
		})

		assert.ErrorIs(t, apperr.ErrNotFound, err)
		assert.Zero(t, num)
	})

	t.Run("should fail if source deployment does not exist", func(t *testing.T) {
		user := authfixture.User()
		target := fixture.Target(fixture.WithTargetCreatedBy(user.ID()))
		app := fixture.App(
			fixture.WithAppCreatedBy(user.ID()),
			fixture.WithEnvironmentConfig(
				domain.NewEnvironmentConfig(target.ID()),
				domain.NewEnvironmentConfig(target.ID()),
			),
		)
		handler, ctx, _ := arrange(t,
			fixture.WithUsers(&user),
			fixture.WithTargets(&target),
			fixture.WithApps(&app),
		)

		num, err := handler(ctx, promote.Command{
			AppID:            string(app.ID()),
			DeploymentNumber: 1,
		})

		assert.ErrorIs(t, apperr.ErrNotFound, err)
		assert.Zero(t, num)
	})

	t.Run("should returns an err if trying to promote a production deployment", func(t *testing.T) {
		user := authfixture.User()
		target := fixture.Target(fixture.WithTargetCreatedBy(user.ID()))
		app := fixture.App(
			fixture.WithAppCreatedBy(user.ID()),
			fixture.WithEnvironmentConfig(
				domain.NewEnvironmentConfig(target.ID()),
				domain.NewEnvironmentConfig(target.ID()),
			),
		)
		deployment := fixture.Deployment(
			fixture.WithDeploymentRequestedBy(user.ID()),
			fixture.FromApp(app),
		)
		handler, ctx, _ := arrange(t,
			fixture.WithUsers(&user),
			fixture.WithTargets(&target),
			fixture.WithApps(&app),
			fixture.WithDeployments(&deployment),
		)

		number, err := handler(ctx, promote.Command{
			AppID:            string(deployment.ID().AppID()),
			DeploymentNumber: int(deployment.ID().DeploymentNumber()),
		})

		assert.ErrorIs(t, domain.ErrCouldNotPromoteProductionDeployment, err)
		assert.Zero(t, number)
	})

	t.Run("should correctly creates a new deployment based on the provided one", func(t *testing.T) {
		user := authfixture.User()
		target := fixture.Target(fixture.WithTargetCreatedBy(user.ID()))
		app := fixture.App(
			fixture.WithAppCreatedBy(user.ID()),
			fixture.WithEnvironmentConfig(
				domain.NewEnvironmentConfig(target.ID()),
				domain.NewEnvironmentConfig(target.ID()),
			),
		)
		deployment := fixture.Deployment(
			fixture.WithDeploymentRequestedBy(user.ID()),
			fixture.FromApp(app),
			fixture.ForEnvironment(domain.Staging),
		)
		handler, ctx, dispatcher := arrange(t,
			fixture.WithUsers(&user),
			fixture.WithTargets(&target),
			fixture.WithApps(&app),
			fixture.WithDeployments(&deployment),
		)

		number, err := handler(ctx, promote.Command{
			AppID:            string(deployment.ID().AppID()),
			DeploymentNumber: int(deployment.ID().DeploymentNumber()),
		})

		assert.Nil(t, err)
		assert.Equal(t, 2, number)
		assert.HasLength(t, 1, dispatcher.Signals())
		created := assert.Is[domain.DeploymentCreated](t, dispatcher.Signals()[0])
		assert.DeepEqual(t, domain.DeploymentCreated{
			ID:        domain.DeploymentIDFrom(app.ID(), 2),
			Config:    created.Config,
			State:     created.State,
			Source:    deployment.Source(),
			Requested: shared.ActionFrom(user.ID(), assert.NotZero(t, created.Requested.At())),
		}, created)
	})
}
