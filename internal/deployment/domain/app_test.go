package domain_test

import (
	"testing"

	auth "github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/deployment/fixture"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/assert"
	shared "github.com/YuukanOO/seelf/pkg/domain"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/must"
)

func Test_App(t *testing.T) {
	t.Run("could be created", func(t *testing.T) {
		t.Run("should require a unique name across both target environments", func(t *testing.T) {
			var (
				appname    domain.AppName = "my-app"
				uid        auth.UserID    = "uid"
				production                = domain.NewEnvironmentConfig("production-target")
				staging                   = domain.NewEnvironmentConfig("staging-target")
			)

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

				assert.ErrorIs(t, test.expected, err)
			}
		})

		t.Run("should correctly creates a new app", func(t *testing.T) {
			var (
				appname    domain.AppName = "my-app"
				uid        auth.UserID    = "uid"
				production                = domain.NewEnvironmentConfig("production-target")
				staging                   = domain.NewEnvironmentConfig("staging-target")
			)

			app, err := domain.NewApp(appname,
				domain.NewEnvironmentConfigRequirement(production, true, true),
				domain.NewEnvironmentConfigRequirement(staging, true, true),
				uid)

			assert.Nil(t, err)
			assert.NotZero(t, app.ID())
			assert.False(t, app.VersionControl().HasValue())

			evt := assert.EventIs[domain.AppCreated](t, &app, 0)

			assert.DeepEqual(t, domain.AppCreated{
				ID:         app.ID(),
				Name:       appname,
				Created:    shared.ActionFrom(uid, assert.NotZero(t, evt.Created.At())),
				Production: evt.Production,
				Staging:    evt.Staging,
			}, evt)
			assert.DeepEqual(t, production, evt.Production.Config())
			assert.DeepEqual(t, staging, evt.Staging.Config())
		})
	})

	t.Run("could have version control configured", func(t *testing.T) {
		t.Run("could have a vcs config attached", func(t *testing.T) {
			vcsConfig := domain.NewVersionControl(must.Panic(domain.UrlFrom("http://somewhere.com")))
			vcsConfig.Authenticated("vcskey")

			app := fixture.App()

			err := app.UseVersionControl(vcsConfig)

			assert.Nil(t, err)
			assert.Equal(t, vcsConfig, app.VersionControl().MustGet())
			assert.HasNEvents(t, 2, &app)
			evt := assert.EventIs[domain.AppVersionControlChanged](t, &app, 1)

			assert.Equal(t, domain.AppVersionControlChanged{
				ID:     app.ID(),
				Config: monad.Value(vcsConfig),
			}, evt)
		})

		t.Run("could have a vcs config removed", func(t *testing.T) {
			app := fixture.App()

			assert.Nil(t, app.RemoveVersionControl())
			assert.HasNEvents(t, 1, &app, "should have nothing new since it didn't have a vcs config initially")

			assert.Nil(t, app.UseVersionControl(domain.NewVersionControl(must.Panic(domain.UrlFrom("http://somewhere.com")))))
			assert.Nil(t, app.RemoveVersionControl())

			assert.HasNEvents(t, 3, &app, "should have 2 new events, one for the config added and one for the config removed")
			evt := assert.EventIs[domain.AppVersionControlChanged](t, &app, 2)
			assert.False(t, evt.Config.HasValue())
		})

		t.Run("raise a VCS configured event only if configs are different", func(t *testing.T) {
			vcsConfig := domain.NewVersionControl(must.Panic(domain.UrlFrom("http://somewhere.com")))
			app := fixture.App()

			assert.Nil(t, app.UseVersionControl(vcsConfig))
			assert.Nil(t, app.UseVersionControl(vcsConfig))

			assert.HasNEvents(t, 2, &app, "should raise an event only once since the configs are equal")

			otherConfig := domain.NewVersionControl(must.Panic(domain.UrlFrom("http://somewhere.else.com")))
			assert.Nil(t, app.UseVersionControl(otherConfig))

			assert.HasNEvents(t, 3, &app, "should raise an event since configs are different")
			evt := assert.EventIs[domain.AppVersionControlChanged](t, &app, 2)

			assert.Equal(t, domain.AppVersionControlChanged{
				ID:     app.ID(),
				Config: monad.Value(otherConfig),
			}, evt)
		})

		t.Run("does not allow to modify the vcs config if the app is marked for deletion", func(t *testing.T) {
			app := fixture.App()
			assert.Nil(t, app.RequestDelete("uid"))

			assert.ErrorIs(t, domain.ErrAppCleanupRequested, app.UseVersionControl(
				domain.NewVersionControl(must.Panic(domain.UrlFrom("http://somewhere.com")))))
			assert.ErrorIs(t, domain.ErrAppCleanupRequested, app.RemoveVersionControl())
		})
	})

	t.Run("could change its environment configurations", func(t *testing.T) {
		t.Run("need the app naming to be available", func(t *testing.T) {
			app := fixture.App()

			err := app.HasProductionConfig(domain.NewEnvironmentConfigRequirement(domain.NewEnvironmentConfig("another-target"), false, false))

			assert.ErrorIs(t, apperr.ErrNotFound, err)
		})

		t.Run("do nothing if config are the same", func(t *testing.T) {
			production := domain.NewEnvironmentConfig("production-target")
			staging := domain.NewEnvironmentConfig("staging-target")
			app := fixture.App(fixture.WithEnvironmentConfig(production, staging))

			assert.Nil(t, app.HasProductionConfig(domain.NewEnvironmentConfigRequirement(production, true, true)))
			assert.Nil(t, app.HasStagingConfig(domain.NewEnvironmentConfigRequirement(staging, true, true)))

			assert.HasNEvents(t, 1, &app, "same configs should not trigger new events")
		})

		t.Run("could update configuration variables if no migration has started and raise the appropriate event", func(t *testing.T) {
			config := domain.NewEnvironmentConfig("production-target")
			app := fixture.App(fixture.WithProductionConfig(config))

			config.HasEnvironmentVariables(domain.ServicesEnv{
				"app": {"DEBUG": "false"},
			})

			assert.Nil(t, app.HasProductionConfig(domain.NewEnvironmentConfigRequirement(config, true, true)))

			assert.HasNEvents(t, 2, &app)
			created := assert.EventIs[domain.AppCreated](t, &app, 0)
			evt := assert.EventIs[domain.AppEnvChanged](t, &app, 1)
			assert.Equal(t, app.ID(), evt.ID)
			assert.Equal(t, created.Production.Since(), evt.Config.Since())
			assert.Equal(t, domain.Production, evt.Environment)
			assert.False(t, evt.Config.Migration().HasValue())
			assert.DeepEqual(t, config, evt.Config.Config())
		})

		t.Run("should start a migration if the target has changed and no migration", func(t *testing.T) {
			app := fixture.App()
			config := domain.NewEnvironmentConfig("updated-target")

			assert.Nil(t, app.HasProductionConfig(domain.NewEnvironmentConfigRequirement(config, true, true)))

			assert.HasNEvents(t, 3, &app)
			created := assert.EventIs[domain.AppCreated](t, &app, 0)
			evt := assert.EventIs[domain.AppEnvChanged](t, &app, 1)
			assert.Equal(t, app.ID(), evt.ID)
			assert.Equal(t, domain.Production, evt.Environment)
			assert.DeepEqual(t, config, evt.Config.Config())
			assert.Equal(t, created.Production.Config().Target(), evt.Config.Migration().MustGet().Target())
			assert.Equal(t, created.Production.Since(), evt.Config.Migration().MustGet().Interval().From())
			migrationStarted := assert.EventIs[domain.AppEnvMigrationStarted](t, &app, 2)
			assert.Equal(t, domain.Production, migrationStarted.Environment)
			assert.Equal(t, created.Production.Config().Target(), migrationStarted.Migration.Target())
			assert.Equal(t, created.Production.Since(), migrationStarted.Migration.Interval().From())
		})

		t.Run("should not allow a target update if a migration is running", func(t *testing.T) {
			app := fixture.App()
			config := domain.NewEnvironmentConfig("updated-target")
			assert.Nil(t, app.HasProductionConfig(domain.NewEnvironmentConfigRequirement(config, true, true)))

			config = domain.NewEnvironmentConfig("new-target")
			err := app.HasProductionConfig(domain.NewEnvironmentConfigRequirement(config, true, true))

			assert.ErrorIs(t, domain.ErrAppEnvironmentMigrationAlreadyRunning, err)
		})

		t.Run("does not allow to modify the environment config if the app is marked for deletion", func(t *testing.T) {
			app := fixture.App()
			assert.Nil(t, app.RequestDelete("uid"))

			assert.ErrorIs(t, domain.ErrAppCleanupRequested, app.HasProductionConfig(domain.NewEnvironmentConfigRequirement(domain.NewEnvironmentConfig("another-target"), true, true)))
			assert.ErrorIs(t, domain.ErrAppCleanupRequested, app.HasStagingConfig(domain.NewEnvironmentConfigRequirement(domain.NewEnvironmentConfig("another-target"), true, true)))
		})
	})

	t.Run("could be marked has deleting", func(t *testing.T) {
		production := domain.NewEnvironmentConfig("production-target")
		staging := domain.NewEnvironmentConfig("staging-target")
		app := fixture.App(fixture.WithEnvironmentConfig(production, staging))

		err := app.RequestDelete("uid")

		assert.Nil(t, err)
		assert.HasNEvents(t, 2, &app)
		requested := assert.EventIs[domain.AppCleanupRequested](t, &app, 1)
		assert.DeepEqual(t, domain.AppCleanupRequested{
			ID:         app.ID(),
			Production: requested.Production,
			Staging:    requested.Staging,
			Requested:  shared.ActionFrom[auth.UserID]("uid", requested.Requested.At()),
		}, requested)
		assert.DeepEqual(t, production, requested.Production.Config())
		assert.DeepEqual(t, staging, requested.Staging.Config())
	})

	t.Run("should returns an error if already marked as being deleted", func(t *testing.T) {
		app := fixture.App()
		assert.Nil(t, app.RequestDelete("uid"))

		assert.ErrorIs(t, domain.ErrAppCleanupRequested, app.RequestDelete("uid"))

		assert.HasNEvents(t, 2, &app, "should raise the event once")
		requested := assert.EventIs[domain.AppCleanupRequested](t, &app, 1)
		assert.DeepEqual(t, domain.AppCleanupRequested{
			ID:         app.ID(),
			Production: requested.Production,
			Staging:    requested.Staging,
			Requested:  shared.ActionFrom[auth.UserID]("uid", requested.Requested.At()),
		}, requested)
	})

	t.Run("should be able to mark resources as cleaned up on specific environment and target", func(t *testing.T) {
		t.Run("should returns an error if the target does not correspond to the application state", func(t *testing.T) {
			app := fixture.App()

			err := app.CleanedUp(domain.Production, "another-target")

			assert.ErrorIs(t, domain.ErrAppEnvironmentTargetInvalid, err)
		})

		t.Run("should returns an error if the environment is invalid", func(t *testing.T) {
			app := fixture.App()

			err := app.CleanedUp("something", "another-target")

			assert.ErrorIs(t, domain.ErrInvalidEnvironmentName, err)
		})

		t.Run("should returns an error if the target correspond to the current one and the application has not been requested for deletion", func(t *testing.T) {
			config := domain.NewEnvironmentConfig("a-target")
			app := fixture.App(fixture.WithProductionConfig(config))

			err := app.CleanedUp(domain.Production, config.Target())

			assert.ErrorIs(t, domain.ErrAppEnvironmentCleanupNotAllowed, err)
		})

		t.Run("should mark the current environment as cleaned up", func(t *testing.T) {
			config := domain.NewEnvironmentConfig("a-target")
			app := fixture.App(fixture.WithProductionConfig(config))
			assert.Nil(t, app.RequestDelete("uid"))

			assert.Nil(t, app.CleanedUp(domain.Production, config.Target()))

			assert.HasNEvents(t, 3, &app)
			cleaned := assert.EventIs[domain.AppEnvCleanedUp](t, &app, 2)
			assert.Equal(t, app.ID(), cleaned.ID)
			assert.Equal(t, config.Target(), cleaned.Target)
			assert.Equal(t, domain.Production, cleaned.Environment)
			assert.True(t, cleaned.Config.IsCleanedUp())
		})

		t.Run("should returns an error if trying to cleanup the same environment twice", func(t *testing.T) {
			config := domain.NewEnvironmentConfig("a-target")
			app := fixture.App(fixture.WithProductionConfig(config))
			assert.Nil(t, app.RequestDelete("uid"))
			assert.Nil(t, app.CleanedUp(domain.Production, config.Target()))

			err := app.CleanedUp(domain.Production, config.Target())

			assert.ErrorIs(t, domain.ErrAppEnvironmentAlreadyCleaned, err)
		})

		t.Run("should end the current migration if the target matches", func(t *testing.T) {
			config := domain.NewEnvironmentConfig("initial-target")
			app := fixture.App(fixture.WithProductionConfig(config))
			newConfig := domain.NewEnvironmentConfig("updated-target")
			assert.Nil(t, app.HasProductionConfig(domain.NewEnvironmentConfigRequirement(newConfig, true, true)))

			assert.Nil(t, app.CleanedUp(domain.Production, config.Target()))
			assert.HasNEvents(t, 4, &app)
			cleaned := assert.EventIs[domain.AppEnvCleanedUp](t, &app, 3)
			assert.Equal(t, app.ID(), cleaned.ID)
			assert.Equal(t, domain.Production, cleaned.Environment)
			assert.Equal(t, config.Target(), cleaned.Target)
			assert.False(t, cleaned.Config.IsCleanedUp())
			assert.False(t, cleaned.Config.Migration().HasValue())
		})

		t.Run("should delete the application if everything has been cleaned up correctly", func(t *testing.T) {
			production := domain.NewEnvironmentConfig("production-target")
			staging := domain.NewEnvironmentConfig("staging-target")
			app := fixture.App(fixture.WithEnvironmentConfig(production, staging))
			updatedProduction := domain.NewEnvironmentConfig("updated-production-target")
			assert.Nil(t, app.HasProductionConfig(domain.NewEnvironmentConfigRequirement(updatedProduction, true, true)))
			assert.Nil(t, app.RequestDelete("uid"))

			assert.Nil(t, app.CleanedUp(domain.Production, production.Target()))
			assert.Nil(t, app.CleanedUp(domain.Staging, staging.Target()))
			assert.Nil(t, app.CleanedUp(domain.Production, updatedProduction.Target()))

			assert.HasNEvents(t, 8, &app)
			deleted := assert.EventIs[domain.AppDeleted](t, &app, 7)
			assert.Equal(t, app.ID(), deleted.ID)
		})
	})
}
