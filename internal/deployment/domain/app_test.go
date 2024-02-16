package domain_test

import (
	"fmt"
	"testing"

	auth "github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/must"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

func Test_App(t *testing.T) {
	var (
		appname    domain.AppName = "my-app"
		uid        auth.UserID    = "uid"
		production                = domain.NewEnvironmentConfig("production-target")
		staging                   = domain.NewEnvironmentConfig("staging-target")
	)

	t.Run("should require a unique name across both target environments", func(t *testing.T) {
		invalidAvailability := []domain.AppNamingAvailability{
			domain.AppNamingTakenInProduction,
			domain.AppNamingTakenInStaging,
			domain.AppNamingProductionTargetNotFound,
			domain.AppNamingStagingTargetNotFound,
		}

		for _, availability := range invalidAvailability {
			_, err := domain.NewApp(appname, production, staging, availability, uid)

			testutil.ErrorIs(t, domain.ErrInvalidAppNaming, err)
		}
	})

	t.Run("should correctly creates a new app", func(t *testing.T) {
		app, err := domain.NewApp(appname, production, staging, domain.AppNamingAvailable, uid)

		testutil.IsNil(t, err)
		testutil.NotEquals(t, "", app.ID())
		testutil.IsFalse(t, app.VCS().HasValue())

		evt := testutil.EventIs[domain.AppCreated](t, &app, 0)

		testutil.Equals(t, app.ID(), evt.ID)
		testutil.Equals(t, evt.Created.By(), uid)
		testutil.IsFalse(t, evt.Created.At().IsZero())
		testutil.Equals(t, appname, evt.Name)
	})

	t.Run("could have a vcs config attached", func(t *testing.T) {
		url := must.Panic(domain.UrlFrom("http://somewhere.com"))
		vcsConfig := domain.NewVCSConfig(url).Authenticated("vcskey")

		app := must.Panic(domain.NewApp(appname, production, staging, domain.AppNamingAvailable, uid))
		app.UseVersionControl(vcsConfig)

		testutil.Equals(t, vcsConfig, app.VCS().MustGet())
		testutil.HasNEvents(t, &app, 2)
		evt := testutil.EventIs[domain.AppVCSConfigured](t, &app, 1)
		testutil.Equals(t, app.ID(), evt.ID)
		testutil.Equals(t, vcsConfig, evt.Config)
	})

	t.Run("could have a vcs config removed", func(t *testing.T) {
		url := must.Panic(domain.UrlFrom("http://somewhere.com"))
		app := must.Panic(domain.NewApp(appname, production, staging, domain.AppNamingAvailable, uid))
		app.RemoveVersionControl()

		testutil.HasNEvents(t, &app, 1)

		app.UseVersionControl(domain.NewVCSConfig(url))
		app.RemoveVersionControl()

		testutil.HasNEvents(t, &app, 3)
		testutil.EventIs[domain.AppVCSRemoved](t, &app, 2)
	})

	t.Run("raise a VCS configured event only if configs are different", func(t *testing.T) {
		url := must.Panic(domain.UrlFrom("http://somewhere.com"))
		vcsConfig := domain.NewVCSConfig(url).Authenticated("vcskey")
		app := must.Panic(domain.NewApp(appname, production, staging, domain.AppNamingAvailable, uid))
		app.UseVersionControl(vcsConfig)
		app.UseVersionControl(vcsConfig)

		testutil.HasNEvents(t, &app, 2)

		anotherUrl, _ := domain.UrlFrom("http://somewhere.else.com")
		otherConfig := domain.NewVCSConfig(anotherUrl)
		app.UseVersionControl(otherConfig)
		testutil.HasNEvents(t, &app, 3)
		evt := testutil.EventIs[domain.AppVCSConfigured](t, &app, 2)
		testutil.Equals(t, otherConfig, evt.Config)
	})

	t.Run("need the app naming to be unique when modifying configuration", func(t *testing.T) {
		app := must.Panic(domain.NewApp(appname, production, staging, domain.AppNamingAvailable, uid))

		err := app.WithProductionConfig(staging, domain.TargetAppNamingTaken)

		testutil.ErrorIs(t, domain.ErrInvalidAppNaming, err)
	})

	t.Run("need the app naming target id to exists when modifying configuration", func(t *testing.T) {
		app := must.Panic(domain.NewApp(appname, production, staging, domain.AppNamingAvailable, uid))

		err := app.WithProductionConfig(staging, domain.TargetAppNamingTargetNotFound)

		testutil.ErrorIs(t, domain.ErrInvalidAppNaming, err)
	})

	t.Run("raise an env changed event only if the new config is different", func(t *testing.T) {
		app := must.Panic(domain.NewApp(appname, production, staging, domain.AppNamingAvailable, uid))

		errProd := app.WithProductionConfig(production, domain.TargetAppNamingAvailable)
		errStaging := app.WithStagingConfig(staging, domain.TargetAppNamingAvailable)

		testutil.IsNil(t, errProd)
		testutil.IsNil(t, errStaging)
		testutil.HasNEvents(t, &app, 1)

		newConfig := domain.
			NewEnvironmentConfig("new-target").
			WithEnvironmentVariables(domain.ServicesEnv{
				"app": {"DEBUG": "true"},
			})

		errProd = app.WithProductionConfig(newConfig, domain.TargetAppNamingAvailable)
		errStaging = app.WithStagingConfig(newConfig, domain.TargetAppNamingAvailable)

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

	t.Run("could be marked for deletion only if not already the case", func(t *testing.T) {
		app := must.Panic(domain.NewApp(appname, production, staging, domain.AppNamingAvailable, uid))

		app.RequestCleanup("uid")
		app.RequestCleanup("uid")

		testutil.HasNEvents(t, &app, 2)
		evt := testutil.EventIs[domain.AppCleanupRequested](t, &app, 1)
		testutil.Equals(t, app.ID(), evt.ID)
		testutil.Equals(t, "uid", evt.Requested.By())
	})

	t.Run("should not allow a deletion if there are running or pending deployments for this app", func(t *testing.T) {
		app := must.Panic(domain.NewApp(appname, production, staging, domain.AppNamingAvailable, uid))

		app.RequestCleanup("uid")

		err := app.Delete(1)

		testutil.ErrorIs(t, domain.ErrAppHasRunningOrPendingDeployments, err)
		testutil.HasNEvents(t, &app, 2)
	})

	t.Run("raise an error if delete is called for a non cleaned up app", func(t *testing.T) {
		app := must.Panic(domain.NewApp(appname, production, staging, domain.AppNamingAvailable, uid))

		err := app.Delete(0)

		testutil.ErrorIs(t, domain.ErrAppCleanupNeeded, err)
	})

	t.Run("could be deleted", func(t *testing.T) {
		app := must.Panic(domain.NewApp(appname, production, staging, domain.AppNamingAvailable, uid))
		app.RequestCleanup("uid")

		err := app.Delete(0)

		testutil.IsNil(t, err)
		testutil.HasNEvents(t, &app, 3)
		evt := testutil.EventIs[domain.AppDeleted](t, &app, 2)
		testutil.Equals(t, app.ID(), evt.ID)
	})
}

func Test_AppNamingAvailability(t *testing.T) {
	t.Run("should be able to return a detailed error", func(t *testing.T) {
		tests := []struct {
			availability domain.AppNamingAvailability
			production   error
			staging      error
		}{
			{domain.AppNamingProductionTargetNotFound, apperr.ErrNotFound, nil},
			{domain.AppNamingStagingTargetNotFound, nil, apperr.ErrNotFound},
			{domain.AppNamingTakenInProduction, domain.ErrInvalidAppNaming, nil},
			{domain.AppNamingTakenInStaging, nil, domain.ErrInvalidAppNaming},
			{domain.AppNamingProductionTargetNotFound | domain.AppNamingTakenInStaging, apperr.ErrNotFound, domain.ErrInvalidAppNaming},
			{domain.AppNamingAvailable, nil, nil},
		}

		for _, test := range tests {
			t.Run(fmt.Sprintf("availability %d", test.availability), func(t *testing.T) {
				testutil.ErrorIs(t, test.production, test.availability.Error(domain.Production))
				testutil.ErrorIs(t, test.staging, test.availability.Error(domain.Staging))
			})
		}
	})
}

func Test_TargetAppNamingAvailability(t *testing.T) {
	t.Run("should be able to return a detailed error", func(t *testing.T) {
		tests := []struct {
			availability domain.TargetAppNamingAvailability
			expected     error
		}{
			{domain.TargetAppNamingTargetNotFound, apperr.ErrNotFound},
			{domain.TargetAppNamingTaken, domain.ErrInvalidAppNaming},
			{domain.TargetAppNamingTargetNotFound | domain.TargetAppNamingTaken, apperr.ErrNotFound},
			{domain.TargetAppNamingAvailable, nil},
		}

		for _, test := range tests {
			t.Run(fmt.Sprintf("availability %d", test.availability), func(t *testing.T) {
				testutil.ErrorIs(t, test.expected, test.availability.Error())
			})
		}
	})
}
