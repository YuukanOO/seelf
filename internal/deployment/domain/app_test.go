package domain_test

import (
	"testing"

	auth "github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/must"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

func Test_App(t *testing.T) {
	var (
		appname             domain.AppName = "my-app"
		uid                 auth.UserID    = "uid"
		production                         = domain.NewEnvironmentConfig("production-target")
		staging                            = domain.NewEnvironmentConfig("staging-target")
		productionAvailable                = domain.NewEnvironmentConfigRequirement(production, true, true)
		stagingAvailable                   = domain.NewEnvironmentConfigRequirement(staging, true, true)
	)

	t.Run("should require a unique name across both target environments", func(t *testing.T) {
		tests := []struct {
			production domain.EnvironmentConfigRequirement
			staging    domain.EnvironmentConfigRequirement
			expected   error
		}{
			{
				domain.NewEnvironmentConfigRequirement(production, false, false),
				domain.NewEnvironmentConfigRequirement(staging, false, false),
				apperr.ErrNotFound,
			},
			{
				domain.NewEnvironmentConfigRequirement(production, true, true),
				domain.NewEnvironmentConfigRequirement(staging, false, false),
				apperr.ErrNotFound,
			},
			{
				domain.NewEnvironmentConfigRequirement(production, true, false),
				domain.NewEnvironmentConfigRequirement(staging, true, true),
				domain.ErrAppNameAlreadyTaken,
			},
			{
				domain.NewEnvironmentConfigRequirement(production, true, true),
				domain.NewEnvironmentConfigRequirement(staging, true, false),
				domain.ErrAppNameAlreadyTaken,
			},
		}

		for _, test := range tests {
			_, err := domain.NewApp(appname, test.production, test.staging, uid)

			testutil.ErrorIs(t, test.expected, err)
		}
	})

	t.Run("should correctly creates a new app", func(t *testing.T) {
		app, err := domain.NewApp(appname, productionAvailable, stagingAvailable, uid)

		testutil.IsNil(t, err)
		testutil.NotEquals(t, "", app.ID())
		testutil.IsFalse(t, app.VersionControl().HasValue())

		evt := testutil.EventIs[domain.AppCreated](t, &app, 0)

		testutil.Equals(t, app.ID(), evt.ID)
		testutil.Equals(t, evt.Created.By(), uid)
		testutil.IsFalse(t, evt.Created.At().IsZero())
		testutil.IsTrue(t, evt.Production.Equals(production))
		testutil.IsTrue(t, evt.Staging.Equals(staging))
		testutil.Equals(t, appname, evt.Name)
	})

	t.Run("could have a vcs config attached", func(t *testing.T) {
		url := must.Panic(domain.UrlFrom("http://somewhere.com"))
		vcsConfig := domain.NewVersionControl(url)
		vcsConfig.Authenticated("vcskey")

		app := must.Panic(domain.NewApp(appname, productionAvailable, stagingAvailable, uid))
		app.UseVersionControl(vcsConfig)

		testutil.Equals(t, vcsConfig, app.VersionControl().MustGet())
		testutil.HasNEvents(t, &app, 2)
		evt := testutil.EventIs[domain.AppVersionControlConfigured](t, &app, 1)
		testutil.Equals(t, app.ID(), evt.ID)
		testutil.Equals(t, vcsConfig, evt.Config)
	})

	t.Run("could have a vcs config removed", func(t *testing.T) {
		url := must.Panic(domain.UrlFrom("http://somewhere.com"))
		app := must.Panic(domain.NewApp(appname, productionAvailable, stagingAvailable, uid))
		app.RemoveVersionControl()

		testutil.HasNEvents(t, &app, 1)

		app.UseVersionControl(domain.NewVersionControl(url))
		app.RemoveVersionControl()

		testutil.HasNEvents(t, &app, 3)
		testutil.EventIs[domain.AppVersionControlRemoved](t, &app, 2)
	})

	t.Run("raise a VCS configured event only if configs are different", func(t *testing.T) {
		url := must.Panic(domain.UrlFrom("http://somewhere.com"))
		vcsConfig := domain.NewVersionControl(url)
		vcsConfig.Authenticated("vcskey")
		app := must.Panic(domain.NewApp(appname, productionAvailable, stagingAvailable, uid))
		app.UseVersionControl(vcsConfig)
		app.UseVersionControl(vcsConfig)

		testutil.HasNEvents(t, &app, 2)

		anotherUrl, _ := domain.UrlFrom("http://somewhere.else.com")
		otherConfig := domain.NewVersionControl(anotherUrl)
		app.UseVersionControl(otherConfig)
		testutil.HasNEvents(t, &app, 3)
		evt := testutil.EventIs[domain.AppVersionControlConfigured](t, &app, 2)
		testutil.Equals(t, otherConfig, evt.Config)
	})

	t.Run("does not allow to modify the vcs config if the app is marked for deletion", func(t *testing.T) {
		url := must.Panic(domain.UrlFrom("http://somewhere.com"))
		app := must.Panic(domain.NewApp(appname, productionAvailable, stagingAvailable, uid))
		app.RequestCleanup("uid")

		testutil.ErrorIs(t, domain.ErrAppCleanupRequested, app.UseVersionControl(domain.NewVersionControl(url)))
		testutil.ErrorIs(t, domain.ErrAppCleanupRequested, app.RemoveVersionControl())
	})

	t.Run("need the app naming to be available when modifying a configuration", func(t *testing.T) {
		app := must.Panic(domain.NewApp(appname, productionAvailable, stagingAvailable, uid))

		err := app.HasProductionConfig(domain.NewEnvironmentConfigRequirement(staging, false, false))

		testutil.ErrorIs(t, apperr.ErrNotFound, err)
	})

	t.Run("should update the environment config version only if target has changed", func(t *testing.T) {
		app := must.Panic(domain.NewApp(appname, productionAvailable, stagingAvailable, uid))

		newConfig := domain.NewEnvironmentConfig(production.Target())
		newConfig.HasEnvironmentVariables(domain.ServicesEnv{
			"app": {"DEBUG": "another value"},
		})

		err := app.HasProductionConfig(domain.NewEnvironmentConfigRequirement(newConfig, true, true))

		testutil.IsNil(t, err)
		testutil.Equals(t, production.Version(), app.Production().Version())

		newConfig = domain.NewEnvironmentConfig("another-target")

		err = app.HasProductionConfig(domain.NewEnvironmentConfigRequirement(newConfig, true, true))

		testutil.IsNil(t, err)
		testutil.NotEquals(t, production.Version(), app.Production().Version())
		testutil.Equals(t, newConfig.Version(), app.Production().Version())
	})

	t.Run("raise an env changed event only if the new config is different", func(t *testing.T) {
		app := must.Panic(domain.NewApp(appname, productionAvailable, stagingAvailable, uid))

		errProd := app.HasProductionConfig(productionAvailable)
		errStaging := app.HasStagingConfig(stagingAvailable)

		testutil.IsNil(t, errProd)
		testutil.IsNil(t, errStaging)
		testutil.HasNEvents(t, &app, 1)

		newConfig := domain.NewEnvironmentConfig("new-target")
		newConfig.HasEnvironmentVariables(domain.ServicesEnv{
			"app": {"DEBUG": "true"},
		})

		errProd = app.HasProductionConfig(domain.NewEnvironmentConfigRequirement(newConfig, true, true))
		errStaging = app.HasStagingConfig(domain.NewEnvironmentConfigRequirement(newConfig, true, true))

		testutil.IsNil(t, errProd)
		testutil.IsNil(t, errStaging)
		testutil.HasNEvents(t, &app, 3)
		evt := testutil.EventIs[domain.AppEnvChanged](t, &app, 1)

		testutil.Equals(t, app.ID(), evt.ID)
		testutil.Equals(t, domain.Production, evt.Environment)
		testutil.DeepEquals(t, newConfig, evt.Config)

		evt = testutil.EventIs[domain.AppEnvChanged](t, &app, 2)

		testutil.Equals(t, app.ID(), evt.ID)
		testutil.Equals(t, domain.Staging, evt.Environment)
		testutil.DeepEquals(t, newConfig, evt.Config)
	})

	t.Run("does not allow to modify the environment config if the app is marked for deletion", func(t *testing.T) {
		app := must.Panic(domain.NewApp(appname, productionAvailable, stagingAvailable, uid))
		app.RequestCleanup("uid")

		testutil.ErrorIs(t, domain.ErrAppCleanupRequested, app.HasProductionConfig(productionAvailable))
		testutil.ErrorIs(t, domain.ErrAppCleanupRequested, app.HasStagingConfig(stagingAvailable))
	})

	t.Run("could be marked for deletion only if not already the case", func(t *testing.T) {
		app := must.Panic(domain.NewApp(appname, productionAvailable, stagingAvailable, uid))

		app.RequestCleanup("uid")
		app.RequestCleanup("uid")

		testutil.HasNEvents(t, &app, 2)
		evt := testutil.EventIs[domain.AppCleanupRequested](t, &app, 1)
		testutil.Equals(t, app.ID(), evt.ID)
		testutil.Equals(t, "uid", evt.Requested.By())
	})

	t.Run("should not allow a deletion if app resources have not been cleaned up", func(t *testing.T) {
		app := must.Panic(domain.NewApp(appname, productionAvailable, stagingAvailable, uid))

		app.RequestCleanup("uid")

		err := app.Delete(false)

		testutil.ErrorIs(t, domain.ErrAppCleanupNeeded, err)
		testutil.HasNEvents(t, &app, 2)
	})

	t.Run("raise an error if delete is called for a non cleaned up app", func(t *testing.T) {
		app := must.Panic(domain.NewApp(appname, productionAvailable, stagingAvailable, uid))

		err := app.Delete(false)

		testutil.ErrorIs(t, domain.ErrAppCleanupNeeded, err)
	})

	t.Run("could be deleted", func(t *testing.T) {
		app := must.Panic(domain.NewApp(appname, productionAvailable, stagingAvailable, uid))
		app.RequestCleanup("uid")

		err := app.Delete(true)

		testutil.IsNil(t, err)
		testutil.HasNEvents(t, &app, 3)
		evt := testutil.EventIs[domain.AppDeleted](t, &app, 2)
		testutil.Equals(t, app.ID(), evt.ID)
	})
}
