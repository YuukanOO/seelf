package domain_test

import (
	"errors"
	"testing"

	auth "github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/deployment/fixture"
	"github.com/YuukanOO/seelf/pkg/assert"
	shared "github.com/YuukanOO/seelf/pkg/domain"
)

func Test_Deployment(t *testing.T) {

	t.Run("should require a version control config to be defined on the app for vcs managed source", func(t *testing.T) {
		app := fixture.App()
		_, err := app.NewDeployment(1, fixture.SourceData(fixture.WithVersionControlNeeded()), domain.Production, "uid")

		assert.ErrorIs(t, domain.ErrVersionControlNotConfigured, err)
	})

	t.Run("should require an app without cleanup requested", func(t *testing.T) {
		app := fixture.App()
		app.RequestCleanup("uid")

		_, err := app.NewDeployment(1, fixture.SourceData(), domain.Production, "uid")

		assert.ErrorIs(t, domain.ErrAppCleanupRequested, err)
	})

	t.Run("should fail for an invalid environment", func(t *testing.T) {
		app := fixture.App()
		_, err := app.NewDeployment(1, fixture.SourceData(), "doesnotexist", "uid")

		assert.ErrorIs(t, domain.ErrInvalidEnvironmentName, err)
	})

	t.Run("should be created from a valid app", func(t *testing.T) {
		config := domain.NewEnvironmentConfig("production-target")
		app := fixture.App(fixture.WithAppName("my-app"), fixture.WithProductionConfig(config))
		sourceData := fixture.SourceData()
		dpl, err := app.NewDeployment(1, sourceData, domain.Production, "uid")
		conf := dpl.Config()

		assert.Nil(t, err)
		assert.Equal(t, domain.DeploymentIDFrom(app.ID(), 1), dpl.ID())
		assert.NotZero(t, dpl.Requested())
		assert.Equal(t, sourceData, dpl.Source())
		assert.Equal(t, app.ID(), conf.AppID())
		assert.Equal(t, "my-app", conf.AppName())
		assert.Equal(t, domain.Production, conf.Environment())
		assert.Equal(t, config.Target(), conf.Target())
		assert.DeepEqual(t, config.Vars(), conf.Vars())

		assert.HasNEvents(t, 1, &dpl)
		evt := assert.EventIs[domain.DeploymentCreated](t, &dpl, 0)

		assert.DeepEqual(t, domain.DeploymentCreated{
			ID:        dpl.ID(),
			Config:    dpl.Config(),
			State:     evt.State,
			Source:    dpl.Source(),
			Requested: shared.ActionFrom[auth.UserID]("uid", assert.NotZero(t, evt.Requested.At())),
		}, evt)

		assert.Equal(t, domain.DeploymentStatusPending, evt.State.Status())
		assert.Zero(t, evt.State.ErrCode())
		assert.False(t, evt.State.Services().HasValue())
	})

	t.Run("could be marked has started", func(t *testing.T) {
		dpl := fixture.Deployment()

		err := dpl.HasStarted()

		assert.Nil(t, err)
		assert.HasNEvents(t, 2, &dpl)
		evt := assert.EventIs[domain.DeploymentStateChanged](t, &dpl, 1)

		assert.Equal(t, dpl.ID(), evt.ID)
		assert.Equal(t, domain.DeploymentStatusRunning, evt.State.Status())
		assert.False(t, evt.State.ErrCode().HasValue())
		assert.False(t, evt.State.Services().HasValue())
		assert.NotZero(t, evt.State.StartedAt())
	})

	t.Run("could be marked has ended with services", func(t *testing.T) {
		deployment := fixture.Deployment()
		builder := deployment.Config().ServicesBuilder()
		builder.AddService("aservice", "an/image")
		services := builder.Services()
		assert.Nil(t, deployment.HasStarted())

		err := deployment.HasEnded(services, nil)

		assert.Nil(t, err)
		assert.HasNEvents(t, 3, &deployment, "should have events related to deployment started and ended")

		evt := assert.EventIs[domain.DeploymentStateChanged](t, &deployment, 1)
		assert.Equal(t, deployment.ID(), evt.ID)
		assert.Equal(t, domain.DeploymentStatusRunning, evt.State.Status())

		evt = assert.EventIs[domain.DeploymentStateChanged](t, &deployment, 2)

		assert.Equal(t, deployment.ID(), evt.ID)
		assert.Equal(t, domain.DeploymentStatusSucceeded, evt.State.Status())
		assert.DeepEqual(t, services, evt.State.Services().MustGet())
	})

	t.Run("should default to a deployment without services if has ended without services nor error", func(t *testing.T) {
		dpl := fixture.Deployment()
		assert.Nil(t, dpl.HasStarted())

		err := dpl.HasEnded(nil, nil)

		assert.Nil(t, err)
		assert.HasNEvents(t, 3, &dpl, "should have events related to deployment started and ended")

		evt := assert.EventIs[domain.DeploymentStateChanged](t, &dpl, 2)

		assert.Equal(t, dpl.ID(), evt.ID)
		assert.Equal(t, domain.DeploymentStatusSucceeded, evt.State.Status())
		assert.True(t, evt.State.Services().HasValue())
	})

	t.Run("could be marked has ended with an error", func(t *testing.T) {
		var (
			err    error
			reason = errors.New("failed reason")
		)

		dpl := fixture.Deployment()
		assert.Nil(t, dpl.HasStarted())

		err = dpl.HasEnded(nil, reason)

		assert.Nil(t, err)
		assert.HasNEvents(t, 3, &dpl, "should have events related to deployment started and ended")

		evt := assert.EventIs[domain.DeploymentStateChanged](t, &dpl, 2)

		assert.Equal(t, dpl.ID(), evt.ID)
		assert.Equal(t, domain.DeploymentStatusFailed, evt.State.Status())
		assert.Equal(t, reason.Error(), evt.State.ErrCode().MustGet())
		assert.False(t, evt.State.Services().HasValue())
	})

	t.Run("could be redeployed", func(t *testing.T) {
		app := fixture.App()
		sourceDeployment := fixture.Deployment(fixture.FromApp(app))

		newDeployment, err := app.Redeploy(sourceDeployment, 2, "another-user")

		assert.Nil(t, err)
		assert.Equal(t, domain.DeploymentIDFrom(app.ID(), sourceDeployment.ID().DeploymentNumber()+1), newDeployment.ID())
		assert.DeepEqual(t, sourceDeployment.Config(), newDeployment.Config())
		assert.Equal(t, sourceDeployment.Source(), newDeployment.Source())
		assert.NotZero(t, newDeployment.Requested())

		evt := assert.EventIs[domain.DeploymentCreated](t, &newDeployment, 0)

		assert.DeepEqual(t, domain.DeploymentCreated{
			ID:        newDeployment.ID(),
			Config:    sourceDeployment.Config(),
			State:     evt.State,
			Source:    sourceDeployment.Source(),
			Requested: shared.ActionFrom[auth.UserID]("another-user", assert.NotZero(t, evt.Requested.At())),
		}, evt)

		assert.Equal(t, domain.DeploymentStatusPending, evt.State.Status())
		assert.Zero(t, evt.State.ErrCode())
		assert.False(t, evt.State.Services().HasValue())

	})

	t.Run("should err if trying to redeploy a deployment on the wrong app", func(t *testing.T) {
		source := fixture.Deployment()
		anotherApp := fixture.App()

		_, err := anotherApp.Redeploy(source, 2, "uid")

		assert.ErrorIs(t, domain.ErrInvalidSourceDeployment, err)
	})

	t.Run("could not promote an already in production deployment", func(t *testing.T) {
		app := fixture.App()
		dpl := fixture.Deployment(fixture.FromApp(app), fixture.ForEnvironment(domain.Production))

		_, err := app.Promote(dpl, 2, "another-user")

		assert.ErrorIs(t, domain.ErrCouldNotPromoteProductionDeployment, err)
	})

	t.Run("should err if trying to promote a deployment on the wrong app", func(t *testing.T) {
		source := fixture.Deployment(fixture.ForEnvironment(domain.Staging))
		anotherApp := fixture.App()

		_, err := anotherApp.Promote(source, 2, "uid")

		assert.ErrorIs(t, domain.ErrInvalidSourceDeployment, err)
	})

	t.Run("could promote a staging deployment", func(t *testing.T) {
		productionConfig := domain.NewEnvironmentConfig("production-target")
		app := fixture.App(fixture.WithProductionConfig(productionConfig))
		sourceDeployment := fixture.Deployment(fixture.FromApp(app), fixture.ForEnvironment(domain.Staging))

		promoted, err := app.Promote(sourceDeployment, 2, "another-user")

		assert.Nil(t, err)
		assert.Equal(t, domain.DeploymentIDFrom(app.ID(), 2), promoted.ID())
		assert.Equal(t, sourceDeployment.Config().AppID(), promoted.Config().AppID())
		assert.Equal(t, sourceDeployment.Config().AppName(), promoted.Config().AppName())
		assert.Equal(t, productionConfig.Target(), promoted.Config().Target())
		assert.Equal(t, domain.Production, promoted.Config().Environment())
		assert.DeepEqual(t, sourceDeployment.Config().Vars(), promoted.Config().Vars())
		assert.Equal(t, sourceDeployment.Source(), promoted.Source())
		assert.NotZero(t, promoted.Requested())

		evt := assert.EventIs[domain.DeploymentCreated](t, &promoted, 0)

		assert.DeepEqual(t, domain.DeploymentCreated{
			ID:        promoted.ID(),
			Config:    promoted.Config(),
			State:     evt.State,
			Source:    sourceDeployment.Source(),
			Requested: shared.ActionFrom[auth.UserID]("another-user", assert.NotZero(t, evt.Requested.At())),
		}, evt)
	})
}

func Test_DeploymentEvents(t *testing.T) {
	t.Run("DeploymentStateChanged should expose a method to check for success state", func(t *testing.T) {
		dpl := fixture.Deployment()
		assert.Nil(t, dpl.HasStarted())
		assert.Nil(t, dpl.HasEnded(nil, nil))

		evt := assert.EventIs[domain.DeploymentStateChanged](t, &dpl, 2)
		assert.True(t, evt.HasSucceeded())

		dpl = fixture.Deployment()
		assert.Nil(t, dpl.HasStarted())
		assert.Nil(t, dpl.HasEnded(nil, errors.New("failed")))

		evt = assert.EventIs[domain.DeploymentStateChanged](t, &dpl, 2)
		assert.False(t, evt.HasSucceeded())
	})
}
