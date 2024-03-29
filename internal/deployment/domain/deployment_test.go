package domain_test

import (
	"errors"
	"testing"

	auth "github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

func Test_Deployment(t *testing.T) {
	var (
		uid        auth.UserID             = "uid"
		number     domain.DeploymentNumber = 1
		vcsMeta                            = meta{true}
		nonVcsMeta                         = meta{false}
	)

	t.Run("should require a version control config to be defined on the app for vcs managed source", func(t *testing.T) {
		app := domain.NewApp("my-app", uid)

		_, err := app.NewDeployment(number, vcsMeta, domain.Production, uid)

		testutil.ErrorIs(t, domain.ErrVCSNotConfigured, err)
	})

	t.Run("should require an app without cleanup requested", func(t *testing.T) {
		app := domain.NewApp("my-app", uid)
		app.RequestCleanup("uid")

		_, err := app.NewDeployment(number, nonVcsMeta, domain.Production, uid)

		testutil.ErrorIs(t, domain.ErrAppCleanupRequested, err)
	})

	t.Run("should be created from a valid app", func(t *testing.T) {
		app := domain.NewApp("my-app", uid)
		dpl, err := app.NewDeployment(number, nonVcsMeta, domain.Production, uid)
		conf := dpl.Config()

		testutil.IsNil(t, err)
		testutil.Equals(t, domain.DeploymentIDFrom(app.ID(), number), dpl.ID())
		testutil.Equals(t, nonVcsMeta, dpl.Source().(meta))
		testutil.Equals(t, "my-app", conf.AppName())
		testutil.Equals(t, domain.Production, conf.Environment())

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

		app := domain.NewApp("my-app", uid)
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
		var (
			err      error
			services domain.Services
		)

		app := domain.NewApp("my-app", uid)
		dpl, _ := app.NewDeployment(number, nonVcsMeta, domain.Production, uid)
		services, _ = services.Internal(dpl.Config(), "aservice", "an/image")
		dpl.HasStarted()

		err = dpl.HasEnded(services, nil)

		testutil.IsNil(t, err)
		testutil.HasNEvents(t, &dpl, 3)
		evt := testutil.EventIs[domain.DeploymentStateChanged](t, &dpl, 2)

		testutil.Equals(t, dpl.ID(), evt.ID)
		testutil.Equals(t, domain.DeploymentStatusSucceeded, evt.State.Status())
		testutil.DeepEquals(t, services, evt.State.Services().MustGet())
	})

	t.Run("should default to a deployment without services if has ended without services nor error", func(t *testing.T) {
		var err error

		app := domain.NewApp("my-app", uid)
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

		app := domain.NewApp("my-app", uid)
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
		app := domain.NewApp("my-app", uid)
		dpl, _ := app.NewDeployment(number, nonVcsMeta, domain.Production, uid)

		redpl, err := app.Redeploy(dpl, 2, "another-user")

		testutil.IsNil(t, err)
		testutil.Equals(t, dpl.Config().Environment(), redpl.Config().Environment())
		testutil.Equals(t, dpl.Source(), redpl.Source())
	})

	t.Run("should err if trying to redeploy a deployment on the wrong app", func(t *testing.T) {
		source, _ := domain.NewApp("an-app", uid).NewDeployment(1, nonVcsMeta, domain.Production, uid)

		_, err := domain.NewApp("my-app", uid).Redeploy(source, 2, "uid")

		testutil.ErrorIs(t, domain.ErrInvalidSourceDeployment, err)
	})

	t.Run("could not promote an already in production deployment", func(t *testing.T) {
		app := domain.NewApp("my-app", uid)
		dpl, _ := app.NewDeployment(number, nonVcsMeta, domain.Production, uid)

		_, err := app.Promote(dpl, 2, "another-user")

		testutil.ErrorIs(t, domain.ErrCouldNotPromoteProductionDeployment, err)
	})

	t.Run("should err if trying to promote a deployment on the wrong app", func(t *testing.T) {
		source, _ := domain.NewApp("an-app", uid).NewDeployment(1, nonVcsMeta, domain.Staging, uid)

		_, err := domain.NewApp("my-app", uid).Promote(source, 2, "uid")

		testutil.ErrorIs(t, domain.ErrInvalidSourceDeployment, err)
	})

	t.Run("could promote a staging deployment", func(t *testing.T) {
		app := domain.NewApp("my-app", uid)
		dpl, _ := app.NewDeployment(number, nonVcsMeta, domain.Staging, uid)

		promoted, err := app.Promote(dpl, 2, "another-user")

		testutil.IsNil(t, err)
		testutil.Equals(t, domain.Production, promoted.Config().Environment())
		testutil.Equals(t, dpl.Source(), promoted.Source())
	})
}

type meta struct {
	isVCS bool
}

func (meta) Kind() string    { return "test" }
func (m meta) NeedVCS() bool { return m.isVCS }
