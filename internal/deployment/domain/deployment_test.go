package domain_test

import (
	"errors"
	"testing"

	auth "github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/must"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

func Test_Deployment(t *testing.T) {
	var (
		appname             domain.AppName          = "my-app"
		production                                  = domain.NewEnvironmentConfig("production-target")
		staging                                     = domain.NewEnvironmentConfig("staging-target")
		productionAvailable                         = domain.NewEnvironmentConfigRequirement(production, true, true)
		stagingAvailable                            = domain.NewEnvironmentConfigRequirement(staging, true, true)
		uid                 auth.UserID             = "uid"
		number              domain.DeploymentNumber = 1
		vcsMeta                                     = meta{true}
		nonVcsMeta                                  = meta{false}
		app                                         = must.Panic(domain.NewApp(appname, productionAvailable, stagingAvailable, uid))
	)

	t.Run("should require a version control config to be defined on the app for vcs managed source", func(t *testing.T) {
		_, err := app.NewDeployment(number, vcsMeta, domain.Production, uid)

		testutil.ErrorIs(t, domain.ErrVersionControlNotConfigured, err)
	})

	t.Run("should require an app without cleanup requested", func(t *testing.T) {
		app := must.Panic(domain.NewApp(appname, productionAvailable, stagingAvailable, uid))
		app.RequestCleanup(uid)

		_, err := app.NewDeployment(number, nonVcsMeta, domain.Production, uid)

		testutil.ErrorIs(t, domain.ErrAppCleanupRequested, err)
	})

	t.Run("should fail for an invalid environment", func(t *testing.T) {
		_, err := app.NewDeployment(number, nonVcsMeta, "doesnotexist", uid)

		testutil.ErrorIs(t, domain.ErrInvalidEnvironmentName, err)
	})

	t.Run("should be created from a valid app", func(t *testing.T) {
		dpl, err := app.NewDeployment(number, nonVcsMeta, domain.Production, uid)
		conf := dpl.Config()

		testutil.IsNil(t, err)
		testutil.Equals(t, domain.DeploymentIDFrom(app.ID(), number), dpl.ID())
		testutil.Equals(t, nonVcsMeta, dpl.Source().(meta))
		testutil.Equals(t, app.ID(), conf.AppID())
		testutil.Equals(t, "my-app", conf.AppName())
		testutil.Equals(t, domain.Production, conf.Environment())
		testutil.Equals(t, production.Target(), conf.Target())
		testutil.DeepEquals(t, production.Vars(), conf.Vars())

		testutil.HasNEvents(t, &dpl, 1)
		evt := testutil.EventIs[domain.DeploymentCreated](t, &dpl, 0)

		testutil.Equals(t, dpl.ID(), evt.ID)
		testutil.Equals(t, dpl.Source(), evt.Source)
		testutil.Equals(t, domain.DeploymentStatusPending, evt.State.Status())
		testutil.IsFalse(t, evt.Requested.At().IsZero())
		testutil.Equals(t, uid, evt.Requested.By())
	})

	t.Run("could be marked has started", func(t *testing.T) {
		var err error

		dpl, err := app.NewDeployment(number, nonVcsMeta, domain.Production, uid)

		testutil.IsNil(t, err)

		err = dpl.HasStarted()

		testutil.IsNil(t, err)
		testutil.HasNEvents(t, &dpl, 2)
		evt := testutil.EventIs[domain.DeploymentStateChanged](t, &dpl, 1)

		testutil.Equals(t, dpl.ID(), evt.ID)
		testutil.Equals(t, domain.DeploymentStatusRunning, evt.State.Status())
	})

	t.Run("could be marked has ended with services", func(t *testing.T) {
		var err error

		dpl := must.Panic(app.NewDeployment(number, nonVcsMeta, domain.Production, uid))
		services := domain.Services{
			dpl.Config().NewService("aservice", "an/image"),
		}

		dpl.HasStarted()

		err = dpl.HasEnded(services, nil)

		testutil.IsNil(t, err)
		testutil.HasNEvents(t, &dpl, 3)
		evt := testutil.EventIs[domain.DeploymentStateChanged](t, &dpl, 1)
		testutil.Equals(t, domain.DeploymentStatusRunning, evt.State.Status())

		evt = testutil.EventIs[domain.DeploymentStateChanged](t, &dpl, 2)

		testutil.Equals(t, dpl.ID(), evt.ID)
		testutil.Equals(t, domain.DeploymentStatusSucceeded, evt.State.Status())
		testutil.DeepEquals(t, services, evt.State.Services().MustGet())
	})

	t.Run("should default to a deployment without services if has ended without services nor error", func(t *testing.T) {
		var err error

		dpl, _ := app.NewDeployment(number, nonVcsMeta, domain.Production, uid)
		dpl.HasStarted()

		err = dpl.HasEnded(nil, nil)

		testutil.IsNil(t, err)
		testutil.HasNEvents(t, &dpl, 3)
		evt := testutil.EventIs[domain.DeploymentStateChanged](t, &dpl, 2)

		testutil.Equals(t, dpl.ID(), evt.ID)
		testutil.Equals(t, domain.DeploymentStatusSucceeded, evt.State.Status())
		testutil.IsTrue(t, evt.State.Services().HasValue())
	})

	t.Run("could be marked has ended with an error", func(t *testing.T) {
		var (
			err    error
			reason = errors.New("failed reason")
		)

		dpl, _ := app.NewDeployment(number, nonVcsMeta, domain.Production, uid)
		dpl.HasStarted()

		err = dpl.HasEnded(nil, reason)

		testutil.IsNil(t, err)
		testutil.HasNEvents(t, &dpl, 3)
		evt := testutil.EventIs[domain.DeploymentStateChanged](t, &dpl, 2)

		testutil.Equals(t, dpl.ID(), evt.ID)
		testutil.Equals(t, domain.DeploymentStatusFailed, evt.State.Status())
		testutil.Equals(t, reason.Error(), evt.State.ErrCode().MustGet())
		testutil.IsFalse(t, evt.State.Services().HasValue())
	})

	t.Run("could be redeployed", func(t *testing.T) {
		dpl := must.Panic(app.NewDeployment(number, nonVcsMeta, domain.Production, uid))

		redpl, err := app.Redeploy(dpl, 2, "another-user")

		testutil.IsNil(t, err)
		testutil.Equals(t, dpl.Config().Environment(), redpl.Config().Environment())
		testutil.Equals(t, dpl.Source(), redpl.Source())
	})

	t.Run("should err if trying to redeploy a deployment on the wrong app", func(t *testing.T) {
		source := must.Panic(app.NewDeployment(1, nonVcsMeta, domain.Production, uid))
		app := must.Panic(domain.NewApp(appname, productionAvailable, stagingAvailable, uid))

		_, err := app.Redeploy(source, 2, "uid")

		testutil.ErrorIs(t, domain.ErrInvalidSourceDeployment, err)
	})

	t.Run("could not promote an already in production deployment", func(t *testing.T) {
		dpl := must.Panic(app.NewDeployment(number, nonVcsMeta, domain.Production, uid))

		_, err := app.Promote(dpl, 2, "another-user")

		testutil.ErrorIs(t, domain.ErrCouldNotPromoteProductionDeployment, err)
	})

	t.Run("should err if trying to promote a deployment on the wrong app", func(t *testing.T) {
		source := must.Panic(app.NewDeployment(1, nonVcsMeta, domain.Staging, uid))
		app := must.Panic(domain.NewApp(appname, productionAvailable, stagingAvailable, uid))

		_, err := app.Promote(source, 2, "uid")

		testutil.ErrorIs(t, domain.ErrInvalidSourceDeployment, err)
	})

	t.Run("could promote a staging deployment", func(t *testing.T) {
		dpl := must.Panic(app.NewDeployment(number, nonVcsMeta, domain.Staging, uid))

		promoted, err := app.Promote(dpl, 2, "another-user")

		testutil.IsNil(t, err)
		testutil.Equals(t, domain.Production, promoted.Config().Environment())
		testutil.Equals(t, dpl.Source(), promoted.Source())
	})
}

func Test_DeploymentEvents(t *testing.T) {
	t.Run("DeploymentStateChanged should expose a method to check for success state", func(t *testing.T) {
		app := must.Panic(domain.NewApp("my-app",
			domain.NewEnvironmentConfigRequirement(domain.NewEnvironmentConfig("production-target"), true, true),
			domain.NewEnvironmentConfigRequirement(domain.NewEnvironmentConfig("staging-target"), true, true),
			"uid",
		))
		dpl := must.Panic(app.NewDeployment(1, meta{}, domain.Staging, "uid"))
		testutil.IsNil(t, dpl.HasStarted())
		testutil.IsNil(t, dpl.HasEnded(nil, nil))

		evt := testutil.EventIs[domain.DeploymentStateChanged](t, &dpl, 2)
		testutil.IsTrue(t, evt.HasSucceeded())

		dpl = must.Panic(app.NewDeployment(2, meta{}, domain.Staging, "uid"))
		testutil.IsNil(t, dpl.HasStarted())
		testutil.IsNil(t, dpl.HasEnded(nil, errors.New("failed")))

		evt = testutil.EventIs[domain.DeploymentStateChanged](t, &dpl, 2)
		testutil.IsFalse(t, evt.HasSucceeded())
	})
}

type meta struct {
	isVCS bool
}

func (meta) Kind() string               { return "test" }
func (m meta) NeedVersionControl() bool { return m.isVCS }
