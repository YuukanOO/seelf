package domain_test

import (
	"testing"

	auth "github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

func Test_App(t *testing.T) {
	t.Run("should correctly creates a new app", func(t *testing.T) {
		var (
			name domain.UniqueAppName = "my-first-app"
			uid  auth.UserID          = "anid"
		)

		app := domain.NewApp(name, uid)

		testutil.NotEquals(t, "", app.ID())
		testutil.IsFalse(t, app.VCS().HasValue())

		evt := testutil.EventIs[domain.AppCreated](t, &app, 0)

		testutil.Equals(t, app.ID(), evt.ID)
		testutil.Equals(t, uid, evt.Created.By())
		testutil.IsFalse(t, evt.Created.At().IsZero())
		testutil.Equals(t, name, evt.Name)
	})

	t.Run("could have a vcs config attached", func(t *testing.T) {
		url, _ := domain.UrlFrom("http://somewhere.com")
		vcsConfig := domain.NewVCSConfig(url).Authenticated("vcskey")

		app := domain.NewApp("an-app", "uid")
		app.UseVersionControl(vcsConfig)

		testutil.Equals(t, vcsConfig, app.VCS().MustGet())
		testutil.HasNEvents(t, &app, 2)
		evt := testutil.EventIs[domain.AppVCSConfigured](t, &app, 1)
		testutil.Equals(t, app.ID(), evt.ID)
		testutil.Equals(t, vcsConfig, evt.Config)
	})

	t.Run("could have a vcs config removed", func(t *testing.T) {
		url, _ := domain.UrlFrom("http://somewhere.com")
		app := domain.NewApp("an-app", "uid")
		app.RemoveVersionControl()

		testutil.HasNEvents(t, &app, 1)

		app.UseVersionControl(domain.NewVCSConfig(url))
		app.RemoveVersionControl()

		testutil.HasNEvents(t, &app, 3)
		testutil.EventIs[domain.AppVCSRemoved](t, &app, 2)
	})

	t.Run("raise a VCS configured event only if configs are different", func(t *testing.T) {
		url, _ := domain.UrlFrom("http://somewhere.com")
		vcsConfig := domain.NewVCSConfig(url).Authenticated("vcskey")
		app := domain.NewApp("an-app", "uid")
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

	t.Run("could have environment variables configured", func(t *testing.T) {
		envs := domain.EnvironmentsEnv{
			domain.Production: {
				"app": {
					"DEBUG": "true",
				},
			},
		}

		app := domain.NewApp("an-app", "uid")
		app.HasEnvironmentVariables(envs)

		testutil.HasNEvents(t, &app, 2)
		evt := testutil.EventIs[domain.AppEnvChanged](t, &app, 1)
		testutil.Equals(t, app.ID(), evt.ID)
		testutil.DeepEquals(t, envs, evt.Env)
	})

	t.Run("could have environment variables removed", func(t *testing.T) {
		app := domain.NewApp("an-app", "uid")
		app.RemoveEnvironmentVariables()

		testutil.HasNEvents(t, &app, 1)

		app.HasEnvironmentVariables(domain.EnvironmentsEnv{
			domain.Production: {
				"app": {
					"DEBUG": "true",
				},
			},
		})
		app.RemoveEnvironmentVariables()

		testutil.HasNEvents(t, &app, 3)
		testutil.EventIs[domain.AppEnvRemoved](t, &app, 2)
	})

	t.Run("raise an env changed event only if values are different", func(t *testing.T) {
		envs := domain.EnvironmentsEnv{
			domain.Production: {
				"app": {
					"DEBUG": "true",
				},
			},
		}

		app := domain.NewApp("an-app", "uid")
		app.HasEnvironmentVariables(envs)
		app.HasEnvironmentVariables(envs)

		testutil.HasNEvents(t, &app, 2)

		otherEnv := domain.EnvironmentsEnv{
			domain.Production: {
				"app": {
					"DEBUG": "false",
				},
			},
		}
		app.HasEnvironmentVariables(otherEnv)

		testutil.HasNEvents(t, &app, 3)
		evt := testutil.EventIs[domain.AppEnvChanged](t, &app, 2)
		testutil.Equals(t, app.ID(), evt.ID)
		testutil.DeepEquals(t, otherEnv, evt.Env)
	})

	t.Run("could be marked for deletion only if not already the case", func(t *testing.T) {
		app := domain.NewApp("an-app", "uid")
		app.RequestCleanup("uid")
		app.RequestCleanup("uid")

		testutil.HasNEvents(t, &app, 2)
		evt := testutil.EventIs[domain.AppCleanupRequested](t, &app, 1)
		testutil.Equals(t, app.ID(), evt.ID)
		testutil.Equals(t, "uid", evt.Requested.By())
	})

	t.Run("should not allow a deletion if there are running or pending deployments for this app", func(t *testing.T) {
		app := domain.NewApp("an-app", "uid")
		app.RequestCleanup("uid")

		err := app.Delete(1)

		testutil.ErrorIs(t, domain.ErrAppHasRunningOrPendingDeployments, err)
		testutil.HasNEvents(t, &app, 2)
	})

	t.Run("raise an error if delete is called for a non cleaned up app", func(t *testing.T) {
		app := domain.NewApp("an-app", "uid")

		err := app.Delete(0)

		testutil.ErrorIs(t, domain.ErrAppCleanupNeeded, err)
	})

	t.Run("could be deleted", func(t *testing.T) {
		app := domain.NewApp("an-app", "uid")
		app.RequestCleanup("uid")
		err := app.Delete(0)

		testutil.IsNil(t, err)
		testutil.HasNEvents(t, &app, 3)
		evt := testutil.EventIs[domain.AppDeleted](t, &app, 2)
		testutil.Equals(t, app.ID(), evt.ID)
	})
}
