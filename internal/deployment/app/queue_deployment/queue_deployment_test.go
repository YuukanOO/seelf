package queue_deployment_test

import (
	"context"
	"testing"

	authfixture "github.com/YuukanOO/seelf/internal/auth/fixture"
	"github.com/YuukanOO/seelf/internal/deployment/app/queue_deployment"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/deployment/fixture"
	"github.com/YuukanOO/seelf/internal/deployment/infra/source/raw"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/assert"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/bus/spy"
	shared "github.com/YuukanOO/seelf/pkg/domain"
	"github.com/YuukanOO/seelf/pkg/validate"
)

func Test_QueueDeployment(t *testing.T) {

	arrange := func(tb testing.TB, seed ...fixture.SeedBuilder) (
		bus.RequestHandler[int, queue_deployment.Command],
		context.Context,
		spy.Dispatcher,
	) {
		context := fixture.PrepareDatabase(tb, seed...)
		return queue_deployment.Handler(context.AppsStore, context.DeploymentsStore, context.DeploymentsStore, raw.New()), context.Context, context.Dispatcher
	}

	t.Run("should fail if the app does not exist", func(t *testing.T) {
		handler, ctx, _ := arrange(t)

		num, err := handler(ctx, queue_deployment.Command{
			AppID:       "does-not-exist",
			Environment: "production",
		})

		assert.ErrorIs(t, apperr.ErrNotFound, err)
		assert.Zero(t, num)
	})

	t.Run("should fail if payload is empty", func(t *testing.T) {
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

		num, err := handler(ctx, queue_deployment.Command{
			AppID:       string(app.ID()),
			Environment: "production",
		})

		assert.ErrorIs(t, domain.ErrInvalidSourcePayload, err)
		assert.Zero(t, num)
	})

	t.Run("should fail if no environment has been given", func(t *testing.T) {
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

		num, err := handler(ctx, queue_deployment.Command{
			AppID: string(app.ID()),
		})

		assert.Zero(t, num)
		assert.ValidationError(t, validate.FieldErrors{
			"environment": domain.ErrInvalidEnvironmentName,
		}, err)
	})

	t.Run("should succeed if everything is good", func(t *testing.T) {
		user := authfixture.User()
		target := fixture.Target(fixture.WithTargetCreatedBy(user.ID()))
		app := fixture.App(
			fixture.WithAppCreatedBy(user.ID()),
			fixture.WithEnvironmentConfig(
				domain.NewEnvironmentConfig(target.ID()),
				domain.NewEnvironmentConfig(target.ID()),
			),
		)
		handler, ctx, dispatcher := arrange(t,
			fixture.WithUsers(&user),
			fixture.WithTargets(&target),
			fixture.WithApps(&app),
		)

		num, err := handler(ctx, queue_deployment.Command{
			AppID:       string(app.ID()),
			Environment: "production",
			Source:      "some-payload",
		})

		assert.Nil(t, err)
		assert.Equal(t, 1, num)
		assert.HasLength(t, 1, dispatcher.Signals())
		created := assert.Is[domain.DeploymentCreated](t, dispatcher.Signals()[0])
		assert.DeepEqual(t, domain.DeploymentCreated{
			ID:        domain.DeploymentIDFrom(app.ID(), 1),
			Config:    created.Config,
			State:     created.State,
			Source:    raw.Data("some-payload"),
			Requested: shared.ActionFrom(user.ID(), assert.NotZero(t, created.Requested.At())),
		}, created)
	})
}
