package domain_test

import (
	"testing"

	auth "github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/deployment/fixture"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/assert"
	shared "github.com/YuukanOO/seelf/pkg/domain"
	"github.com/YuukanOO/seelf/pkg/must"
)

func Test_App(t *testing.T) {

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
			Production: production,
			Staging:    staging,
		}, evt)
	})

	t.Run("could have a vcs config attached", func(t *testing.T) {
		vcsConfig := domain.NewVersionControl(must.Panic(domain.UrlFrom("http://somewhere.com")))
		vcsConfig.Authenticated("vcskey")

		app := fixture.App()

		err := app.UseVersionControl(vcsConfig)

		assert.Nil(t, err)
		assert.Equal(t, vcsConfig, app.VersionControl().MustGet())
		assert.HasNEvents(t, 2, &app)
		evt := assert.EventIs[domain.AppVersionControlConfigured](t, &app, 1)

		assert.Equal(t, domain.AppVersionControlConfigured{
			ID:     app.ID(),
			Config: vcsConfig,
		}, evt)
	})

	t.Run("could have a vcs config removed", func(t *testing.T) {
		app := fixture.App()

		assert.Nil(t, app.RemoveVersionControl())
		assert.HasNEvents(t, 1, &app, "should have nothing new since it didn't have a vcs config initially")

		assert.Nil(t, app.UseVersionControl(domain.NewVersionControl(must.Panic(domain.UrlFrom("http://somewhere.com")))))
		assert.Nil(t, app.RemoveVersionControl())

		assert.HasNEvents(t, 3, &app, "should have 2 new events, one for the config added and one for the config removed")
		assert.EventIs[domain.AppVersionControlRemoved](t, &app, 2)
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
		evt := assert.EventIs[domain.AppVersionControlConfigured](t, &app, 2)

		assert.Equal(t, domain.AppVersionControlConfigured{
			ID:     app.ID(),
			Config: otherConfig,
		}, evt)
	})

	t.Run("does not allow to modify the vcs config if the app is marked for deletion", func(t *testing.T) {
		app := fixture.App()
		app.RequestCleanup("uid")

		assert.ErrorIs(t, domain.ErrAppCleanupRequested, app.UseVersionControl(
			domain.NewVersionControl(must.Panic(domain.UrlFrom("http://somewhere.com")))))
		assert.ErrorIs(t, domain.ErrAppCleanupRequested, app.RemoveVersionControl())
	})

	t.Run("need the app naming to be available when modifying a configuration", func(t *testing.T) {
		app := fixture.App()

		err := app.HasProductionConfig(domain.NewEnvironmentConfigRequirement(domain.NewEnvironmentConfig("another-target"), false, false))

		assert.ErrorIs(t, apperr.ErrNotFound, err)
	})

	t.Run("should update the environment config version only if target has changed", func(t *testing.T) {
		config := domain.NewEnvironmentConfig("production-target")
		app := fixture.App(fixture.WithEnvironmentConfig(config, config))

		newConfig := domain.NewEnvironmentConfig(config.Target())
		newConfig.HasEnvironmentVariables(domain.ServicesEnv{
			"app": {"DEBUG": "another value"},
		})

		assert.Nil(t, app.HasProductionConfig(domain.NewEnvironmentConfigRequirement(newConfig, true, true)))
		changed := assert.EventIs[domain.AppEnvChanged](t, &app, 1)

		assert.Equal(t, changed.OldConfig.Version(), changed.Config.Version(), "same target should keep the same version")

		newConfig = domain.NewEnvironmentConfig("another-target")

		assert.Nil(t, app.HasProductionConfig(domain.NewEnvironmentConfigRequirement(newConfig, true, true)))
		changed = assert.EventIs[domain.AppEnvChanged](t, &app, 2)

		assert.NotEqual(t, changed.OldConfig.Version(), changed.Config.Version())
		assert.Equal(t, newConfig.Version(), changed.Config.Version(), "should match the new config version")
	})

	t.Run("raise an env changed event only if the new config is different", func(t *testing.T) {
		production := domain.NewEnvironmentConfig("production-target")
		staging := domain.NewEnvironmentConfig("staging-target")
		app := fixture.App(fixture.WithEnvironmentConfig(production, staging))

		assert.Nil(t, app.HasProductionConfig(domain.NewEnvironmentConfigRequirement(production, true, true)))
		assert.Nil(t, app.HasStagingConfig(domain.NewEnvironmentConfigRequirement(staging, true, true)))

		assert.HasNEvents(t, 1, &app, "same configs should not trigger new events")

		newConfig := domain.NewEnvironmentConfig("new-target")
		newConfig.HasEnvironmentVariables(domain.ServicesEnv{
			"app": {"DEBUG": "true"},
		})

		assert.Nil(t, app.HasProductionConfig(domain.NewEnvironmentConfigRequirement(newConfig, true, true)))
		assert.Nil(t, app.HasStagingConfig(domain.NewEnvironmentConfigRequirement(newConfig, true, true)))

		assert.HasNEvents(t, 3, &app, "new configs should trigger new events")
		changed := assert.EventIs[domain.AppEnvChanged](t, &app, 1)

		assert.DeepEqual(t, domain.AppEnvChanged{
			ID:          app.ID(),
			Environment: domain.Production,
			Config:      newConfig,
			OldConfig:   production,
		}, changed)

		changed = assert.EventIs[domain.AppEnvChanged](t, &app, 2)

		assert.DeepEqual(t, domain.AppEnvChanged{
			ID:          app.ID(),
			Environment: domain.Staging,
			Config:      newConfig,
			OldConfig:   staging,
		}, changed)
	})

	t.Run("does not allow to modify the environment config if the app is marked for deletion", func(t *testing.T) {
		app := fixture.App()
		app.RequestCleanup("uid")

		assert.ErrorIs(t, domain.ErrAppCleanupRequested, app.HasProductionConfig(domain.NewEnvironmentConfigRequirement(domain.NewEnvironmentConfig("another-target"), true, true)))
		assert.ErrorIs(t, domain.ErrAppCleanupRequested, app.HasStagingConfig(domain.NewEnvironmentConfigRequirement(domain.NewEnvironmentConfig("another-target"), true, true)))
	})

	t.Run("could be marked for deletion only if not already the case", func(t *testing.T) {
		production := domain.NewEnvironmentConfig("production-target")
		staging := domain.NewEnvironmentConfig("staging-target")
		app := fixture.App(fixture.WithEnvironmentConfig(production, staging))

		app.RequestCleanup("uid")
		app.RequestCleanup("uid")

		assert.HasNEvents(t, 2, &app, "should raise the event once")
		evt := assert.EventIs[domain.AppCleanupRequested](t, &app, 1)

		assert.DeepEqual(t, domain.AppCleanupRequested{
			ID:               app.ID(),
			ProductionConfig: production,
			StagingConfig:    staging,
			Requested:        shared.ActionFrom[auth.UserID]("uid", evt.Requested.At()),
		}, evt)
	})

	t.Run("should not allow a deletion if app resources have not been cleaned up", func(t *testing.T) {
		app := fixture.App()
		app.RequestCleanup("uid")

		err := app.Delete(false)

		assert.ErrorIs(t, domain.ErrAppCleanupNeeded, err)
		assert.HasNEvents(t, 2, &app)
	})

	t.Run("raise an error if delete is called for a non cleaned up app", func(t *testing.T) {
		app := fixture.App()

		err := app.Delete(false)

		assert.ErrorIs(t, domain.ErrAppCleanupNeeded, err)
	})

	t.Run("could be deleted", func(t *testing.T) {
		app := fixture.App()
		app.RequestCleanup("uid")

		err := app.Delete(true)

		assert.Nil(t, err)
		assert.HasNEvents(t, 3, &app)
		evt := assert.EventIs[domain.AppDeleted](t, &app, 2)

		assert.Equal(t, domain.AppDeleted{
			ID: app.ID(),
		}, evt)
	})
}

func Test_AppEvents(t *testing.T) {
	t.Run("AppEnvChanged should provide a function to check for target changes", func(t *testing.T) {
		evt := domain.AppEnvChanged{
			ID:          "app",
			Environment: domain.Production,
			Config:      domain.NewEnvironmentConfig("target"),
			OldConfig:   domain.NewEnvironmentConfig("target"),
		}

		assert.False(t, evt.TargetHasChanged())

		evt.OldConfig = domain.NewEnvironmentConfig("another-target")

		assert.True(t, evt.TargetHasChanged())
	})
}
