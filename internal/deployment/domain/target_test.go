package domain_test

import (
	"errors"
	"testing"
	"time"

	auth "github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/deployment/fixture"
	"github.com/YuukanOO/seelf/pkg/assert"
	shared "github.com/YuukanOO/seelf/pkg/domain"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/must"
)

func Test_Target(t *testing.T) {

	t.Run("should fail if the url is not unique", func(t *testing.T) {
		_, err := domain.NewTarget("target",
			domain.NewTargetUrlRequirement(must.Panic(domain.UrlFrom("http://my-url.com")), false),
			domain.NewProviderConfigRequirement(fixture.ProviderConfig(), true), "uid")

		assert.ErrorIs(t, domain.ErrUrlAlreadyTaken, err)
	})

	t.Run("should fail if the config is not unique", func(t *testing.T) {
		_, err := domain.NewTarget("target",
			domain.NewTargetUrlRequirement(must.Panic(domain.UrlFrom("http://my-url.com")), true),
			domain.NewProviderConfigRequirement(fixture.ProviderConfig(), false), "uid")

		assert.ErrorIs(t, domain.ErrConfigAlreadyTaken, err)
	})

	t.Run("should be instantiable", func(t *testing.T) {
		url := must.Panic(domain.UrlFrom("http://my-url.com"))
		config := fixture.ProviderConfig()

		target, err := domain.NewTarget("target",
			domain.NewTargetUrlRequirement(url, true),
			domain.NewProviderConfigRequirement(config, true),
			"uid")

		assert.Nil(t, err)
		assert.HasNEvents(t, 1, &target)
		created := assert.EventIs[domain.TargetCreated](t, &target, 0)

		assert.DeepEqual(t, domain.TargetCreated{
			ID:          assert.NotZero(t, target.ID()),
			Name:        "target",
			Url:         url,
			Provider:    config,
			State:       created.State,
			Entrypoints: make(domain.TargetEntrypoints),
			Created:     shared.ActionFrom[auth.UserID]("uid", assert.NotZero(t, created.Created.At())),
		}, created)

		assert.Equal(t, domain.TargetStatusConfiguring, created.State.Status())
		assert.NotZero(t, created.State.Version())
	})

	t.Run("could be renamed and raise the event only if different", func(t *testing.T) {
		target := fixture.Target(fixture.WithTargetName("old-name"))

		err := target.Rename("new-name")

		assert.Nil(t, err)
		evt := assert.EventIs[domain.TargetRenamed](t, &target, 1)

		assert.Equal(t, domain.TargetRenamed{
			ID:   target.ID(),
			Name: "new-name",
		}, evt)

		assert.Nil(t, target.Rename("new-name"))
		assert.HasNEvents(t, 2, &target, "should have raised the event once")
	})

	t.Run("could not be renamed if delete requested", func(t *testing.T) {
		target := fixture.Target()
		target.Configured(target.CurrentVersion(), nil, nil)

		assert.Nil(t, target.RequestCleanup(false, "uid"))

		assert.ErrorIs(t, domain.ErrTargetCleanupRequested, target.Rename("new-name"))
	})

	t.Run("could have its domain changed if available and raise the event only if different", func(t *testing.T) {
		target := fixture.Target()
		newUrl := must.Panic(domain.UrlFrom("http://new-url.com"))

		err := target.HasUrl(domain.NewTargetUrlRequirement(newUrl, false))
		assert.ErrorIs(t, domain.ErrUrlAlreadyTaken, err)

		err = target.HasUrl(domain.NewTargetUrlRequirement(newUrl, true))
		assert.Nil(t, err)
		evt := assert.EventIs[domain.TargetUrlChanged](t, &target, 1)
		assert.Equal(t, newUrl, evt.Url)

		evtTargetChanged := assert.EventIs[domain.TargetStateChanged](t, &target, 2)
		assert.Equal(t, domain.TargetStatusConfiguring, evtTargetChanged.State.Status())

		assert.Nil(t, target.HasUrl(domain.NewTargetUrlRequirement(newUrl, true)))
		assert.HasNEvents(t, 3, &target)
	})

	t.Run("could not have its domain changed if delete requested", func(t *testing.T) {
		target := fixture.Target()
		target.Configured(target.CurrentVersion(), nil, nil)
		assert.Nil(t, target.RequestCleanup(false, "uid"))

		err := target.HasUrl(domain.NewTargetUrlRequirement(must.Panic(domain.UrlFrom("http://new-url.com")), true))

		assert.ErrorIs(t, domain.ErrTargetCleanupRequested, err)
	})

	t.Run("should forbid a provider change if the fingerprint has changed", func(t *testing.T) {
		target := fixture.Target(fixture.WithProviderConfig(fixture.ProviderConfig(fixture.WithFingerprint("docker"))))

		err := target.HasProvider(domain.NewProviderConfigRequirement(fixture.ProviderConfig(), true))

		assert.ErrorIs(t, domain.ErrTargetProviderUpdateNotPermitted, err)
	})

	t.Run("could have its provider changed if available and raise the event only if different", func(t *testing.T) {
		config := fixture.ProviderConfig(fixture.WithFingerprint("docker"))
		target := fixture.Target(fixture.WithProviderConfig(config))
		newConfig := fixture.ProviderConfig(fixture.WithFingerprint("docker"))

		err := target.HasProvider(domain.NewProviderConfigRequirement(newConfig, false))

		assert.ErrorIs(t, domain.ErrConfigAlreadyTaken, err)

		err = target.HasProvider(domain.NewProviderConfigRequirement(newConfig, true))

		assert.Nil(t, err)
		evt := assert.EventIs[domain.TargetProviderChanged](t, &target, 1)
		assert.Equal(t, newConfig, evt.Provider)

		evtTargetChanged := assert.EventIs[domain.TargetStateChanged](t, &target, 2)
		assert.Equal(t, domain.TargetStatusConfiguring, evtTargetChanged.State.Status())

		assert.Nil(t, target.HasProvider(domain.NewProviderConfigRequirement(newConfig, true)))
		assert.HasNEvents(t, 3, &target, "should raise the event only once")
	})

	t.Run("could not have its provider changed if delete requested", func(t *testing.T) {
		config := fixture.ProviderConfig(fixture.WithFingerprint("docker"))
		target := fixture.Target(fixture.WithProviderConfig(config))
		target.Configured(target.CurrentVersion(), nil, nil)

		assert.Nil(t, target.RequestCleanup(false, "uid"))
		assert.ErrorIs(t, domain.ErrTargetCleanupRequested,
			target.HasProvider(domain.NewProviderConfigRequirement(fixture.ProviderConfig(fixture.WithFingerprint("docker")), true)))
	})

	t.Run("could be marked as configured and raise the appropriate event", func(t *testing.T) {
		target := fixture.Target()

		target.Configured(target.CurrentVersion().Add(-1*time.Hour), nil, nil)

		assert.HasNEvents(t, 1, &target, "should not raise a new event since the version does not match")
		assert.EventIs[domain.TargetCreated](t, &target, 0)

		target.Configured(target.CurrentVersion(), nil, nil)
		target.Configured(target.CurrentVersion(), nil, nil) // Should not raise a new event

		assert.HasNEvents(t, 2, &target, "should raise the event once")
		changed := assert.EventIs[domain.TargetStateChanged](t, &target, 1)
		assert.Equal(t, domain.TargetStatusReady, changed.State.Status())
	})

	t.Run("should handle entrypoints assignment on configuration", func(t *testing.T) {
		target := fixture.Target()
		deployment := fixture.Deployment()

		// Assigning non existing entrypoints should just be ignored
		target.Configured(target.CurrentVersion(), domain.TargetEntrypointsAssigned{
			deployment.ID().AppID(): {
				domain.Production: {
					"non-existing-entrypoint": 5432,
				},
			},
		}, nil)

		assert.HasNEvents(t, 2, &target)
		assert.DeepEqual(t, domain.TargetEntrypoints{}, target.CustomEntrypoints())

		dbService := deployment.Config().NewService("db", "postgres:14-alpine")
		http := dbService.AddHttpEntrypoint(deployment.Config(), 80, domain.HttpEntrypointOptions{})
		tcp := dbService.AddTCPEntrypoint(5432)

		target.ExposeEntrypoints(deployment.ID().AppID(), domain.Production, domain.Services{dbService})

		// Assigning but with an error should ignore new entrypoints
		target.Configured(target.CurrentVersion(), domain.TargetEntrypointsAssigned{
			deployment.ID().AppID(): {
				domain.Production: {
					http.Name(): 8081,
					tcp.Name():  8082,
				},
			},
		}, errors.New("some error"))

		assert.HasNEvents(t, 5, &target)
		assert.DeepEqual(t, domain.TargetEntrypoints{
			deployment.ID().AppID(): {
				domain.Production: {
					http.Name(): monad.None[domain.Port](),
					tcp.Name():  monad.None[domain.Port](),
				},
			},
		}, target.CustomEntrypoints())

		assert.Nil(t, target.Reconfigure())

		// No error, should update the entrypoints correctly
		target.Configured(target.CurrentVersion(), domain.TargetEntrypointsAssigned{
			deployment.ID().AppID(): {
				domain.Production: {
					http.Name():               8081,
					tcp.Name():                8082,
					"non-existing-entrypoint": 5432,
				},
				"non-existing-env": {
					"non-existing-entrypoint": 5432,
				},
			},
			"another-app": {
				"non-existing-env": {
					"non-existing-entrypoint": 5432,
				},
			},
		}, nil)

		assert.HasNEvents(t, 8, &target)
		assert.EventIs[domain.TargetEntrypointsChanged](t, &target, 6)
		changed := assert.EventIs[domain.TargetStateChanged](t, &target, 7)
		assert.Equal(t, domain.TargetStatusReady, changed.State.Status())
		assert.DeepEqual(t, domain.TargetEntrypoints{
			deployment.ID().AppID(): {
				domain.Production: {
					http.Name(): monad.Value[domain.Port](8081),
					tcp.Name():  monad.Value[domain.Port](8082),
				},
			},
		}, target.CustomEntrypoints())
	})

	t.Run("should be able to unexpose entrypoints for a specific app", func(t *testing.T) {
		target := fixture.Target()
		deployment := fixture.Deployment()
		dbService := deployment.Config().NewService("db", "postgres:14-alpine")
		http := dbService.AddHttpEntrypoint(deployment.Config(), 80, domain.HttpEntrypointOptions{})
		tcp := dbService.AddTCPEntrypoint(5432)

		target.UnExposeEntrypoints(deployment.ID().AppID())

		assert.HasNEvents(t, 1, &target, "should not raise an event since no entrypoints were exposed")

		target.ExposeEntrypoints(deployment.ID().AppID(), domain.Production, domain.Services{dbService})
		target.Configured(target.CurrentVersion(), domain.TargetEntrypointsAssigned{
			deployment.ID().AppID(): {
				domain.Production: {
					http.Name(): 8081,
					tcp.Name():  8082,
				},
			},
		}, nil)

		target.UnExposeEntrypoints(deployment.ID().AppID())

		assert.HasNEvents(t, 7, &target)
		assert.DeepEqual(t, domain.TargetEntrypoints{}, target.CustomEntrypoints())
		changed := assert.EventIs[domain.TargetStateChanged](t, &target, 6)
		assert.Equal(t, domain.TargetStatusConfiguring, changed.State.Status())

		target.ExposeEntrypoints(deployment.ID().AppID(), domain.Production, domain.Services{dbService})
		target.Configured(target.CurrentVersion(), domain.TargetEntrypointsAssigned{
			deployment.ID().AppID(): {
				domain.Production: {
					http.Name(): 8081,
					tcp.Name():  8082,
				},
			},
		}, nil)

		target.UnExposeEntrypoints(deployment.ID().AppID(), domain.Staging)
		target.UnExposeEntrypoints(deployment.ID().AppID(), domain.Production)

		assert.HasNEvents(t, 13, &target)
		assert.DeepEqual(t, domain.TargetEntrypoints{}, target.CustomEntrypoints())
		changed = assert.EventIs[domain.TargetStateChanged](t, &target, 12)
		assert.Equal(t, domain.TargetStatusConfiguring, changed.State.Status())
	})

	t.Run("could expose its availability based on its internal state", func(t *testing.T) {
		target := fixture.Target()

		// Configuring
		err := target.CheckAvailability()

		assert.ErrorIs(t, domain.ErrTargetConfigurationInProgress, err)

		// Configuration failed
		target.Configured(target.CurrentVersion(), nil, errors.New("configuration failed"))

		err = target.CheckAvailability()

		assert.ErrorIs(t, domain.ErrTargetConfigurationFailed, err)

		// Configuration success
		assert.Nil(t, target.Reconfigure())

		target.Configured(target.CurrentVersion(), nil, nil)

		err = target.CheckAvailability()

		assert.Nil(t, err)

		// Delete requested
		assert.Nil(t, target.RequestCleanup(false, "uid"))

		err = target.CheckAvailability()

		assert.ErrorIs(t, domain.ErrTargetCleanupRequested, err)
	})

	t.Run("could not be reconfigured if cleanup requested", func(t *testing.T) {
		target := fixture.Target()
		target.Configured(target.CurrentVersion(), nil, nil)
		assert.Nil(t, target.RequestCleanup(false, "uid"))

		assert.ErrorIs(t, domain.ErrTargetCleanupRequested, target.Reconfigure())
	})

	t.Run("could not be reconfigured if configuring", func(t *testing.T) {
		target := fixture.Target()

		assert.ErrorIs(t, domain.ErrTargetConfigurationInProgress, target.Reconfigure())
	})

	t.Run("should not be removed if still used by an app", func(t *testing.T) {
		target := fixture.Target()
		target.Configured(target.CurrentVersion(), nil, nil)

		assert.ErrorIs(t, domain.ErrTargetInUse, target.RequestCleanup(true, "uid"))
	})

	t.Run("should not be removed if configuring", func(t *testing.T) {
		target := fixture.Target()

		assert.ErrorIs(t, domain.ErrTargetConfigurationInProgress, target.RequestCleanup(false, "uid"))
	})

	t.Run("could be removed if no app is using it", func(t *testing.T) {
		target := fixture.Target()
		target.Configured(target.CurrentVersion(), nil, nil)

		err := target.RequestCleanup(false, "uid")

		assert.Nil(t, err)
		assert.HasNEvents(t, 3, &target)
		evt := assert.EventIs[domain.TargetCleanupRequested](t, &target, 2)

		assert.Equal(t, domain.TargetCleanupRequested{
			ID:        target.ID(),
			Requested: shared.ActionFrom[auth.UserID]("uid", assert.NotZero(t, evt.Requested.At())),
		}, evt)
	})

	t.Run("should not raise an event if the target is already marked has deleting", func(t *testing.T) {
		target := fixture.Target()
		target.Configured(target.CurrentVersion(), nil, nil)

		assert.Nil(t, target.RequestCleanup(false, "uid"))
		assert.Nil(t, target.RequestCleanup(false, "uid"))

		assert.HasNEvents(t, 3, &target)
	})

	t.Run("should returns an err if trying to cleanup a target while configuring", func(t *testing.T) {
		target := fixture.Target()

		_, err := target.CleanupStrategy(false)

		assert.ErrorIs(t, domain.ErrTargetConfigurationInProgress, err)
	})

	t.Run("should returns an err if trying to cleanup a target while deployments are still running", func(t *testing.T) {
		target := fixture.Target()
		target.Configured(target.CurrentVersion(), nil, nil)

		_, err := target.CleanupStrategy(true)

		assert.ErrorIs(t, domain.ErrRunningOrPendingDeployments, err)
	})

	t.Run("should returns the skip cleanup strategy if the configuration has failed and the target could not be updated anymore", func(t *testing.T) {
		target := fixture.Target()
		target.Configured(target.CurrentVersion(), nil, nil)
		assert.Nil(t, target.Reconfigure())
		target.Configured(target.CurrentVersion(), nil, errors.New("configuration failed"))
		assert.Nil(t, target.RequestCleanup(false, "uid"))

		s, err := target.CleanupStrategy(false)

		assert.Nil(t, err)
		assert.Equal(t, domain.CleanupStrategySkip, s)
	})

	t.Run("should returns the skip cleanup strategy if the configuration has failed and has never been reachable", func(t *testing.T) {
		target := fixture.Target()
		target.Configured(target.CurrentVersion(), nil, errors.New("configuration failed"))

		s, err := target.CleanupStrategy(false)

		assert.Nil(t, err)
		assert.Equal(t, domain.CleanupStrategySkip, s)
	})

	t.Run("should returns an err if the configuration has failed but the target is still updatable", func(t *testing.T) {
		target := fixture.Target()
		target.Configured(target.CurrentVersion(), nil, nil)
		assert.Nil(t, target.Reconfigure())
		target.Configured(target.CurrentVersion(), nil, errors.New("configuration failed"))

		_, err := target.CleanupStrategy(false)

		assert.ErrorIs(t, domain.ErrTargetConfigurationFailed, err)
	})

	t.Run("should returns the default strategy if the target is correctly configured", func(t *testing.T) {
		target := fixture.Target()
		target.Configured(target.CurrentVersion(), nil, nil)

		s, err := target.CleanupStrategy(false)

		assert.Nil(t, err)
		assert.Equal(t, domain.CleanupStrategyDefault, s)
	})

	t.Run("returns an err if trying to cleanup an app while configuring", func(t *testing.T) {
		target := fixture.Target()

		_, err := target.AppCleanupStrategy(false, true)

		assert.ErrorIs(t, domain.ErrTargetConfigurationInProgress, err)
	})

	t.Run("returns a skip strategy when trying to cleanup an app on a deleting target", func(t *testing.T) {
		target := fixture.Target()
		target.Configured(target.CurrentVersion(), nil, nil)
		assert.Nil(t, target.RequestCleanup(false, "uid"))

		s, err := target.AppCleanupStrategy(false, false)

		assert.Nil(t, err)
		assert.Equal(t, domain.CleanupStrategySkip, s)
	})

	t.Run("returns a skip strategy when trying to cleanup an app when no successful deployment has been made", func(t *testing.T) {
		target := fixture.Target()

		s, err := target.AppCleanupStrategy(false, false)

		assert.Nil(t, err)
		assert.Equal(t, domain.CleanupStrategySkip, s)
	})

	t.Run("returns an error when trying to cleanup an app on a failed target", func(t *testing.T) {
		target := fixture.Target()
		target.Configured(target.CurrentVersion(), nil, nil)
		assert.Nil(t, target.Reconfigure())
		target.Configured(target.CurrentVersion(), nil, errors.New("configuration failed"))

		_, err := target.AppCleanupStrategy(false, true)

		assert.ErrorIs(t, domain.ErrTargetConfigurationFailed, err)
	})

	t.Run("returns an error when trying to cleanup an app but there are still running or pending deployments", func(t *testing.T) {
		target := fixture.Target()
		target.Configured(target.CurrentVersion(), nil, nil)

		_, err := target.AppCleanupStrategy(true, false)

		assert.ErrorIs(t, domain.ErrRunningOrPendingDeployments, err)
	})

	t.Run("returns a default strategy when trying to remove an app and everything is good to process it", func(t *testing.T) {
		target := fixture.Target()
		target.Configured(target.CurrentVersion(), nil, nil)

		s, err := target.AppCleanupStrategy(false, true)

		assert.Nil(t, err)
		assert.Equal(t, domain.CleanupStrategyDefault, s)
	})

	t.Run("should do nothing if trying to expose an empty entrypoints array", func(t *testing.T) {
		target := fixture.Target()

		target.ExposeEntrypoints("appid", domain.Production, domain.Services{})
		assert.HasNEvents(t, 1, &target)

		target.ExposeEntrypoints("appid", domain.Production, nil)
		assert.HasNEvents(t, 1, &target)
	})

	t.Run("should switch to the configuring state if adding new entrypoints to expose", func(t *testing.T) {
		target := fixture.Target()
		deployment := fixture.Deployment()
		appService := deployment.Config().NewService("app", "")
		http := appService.AddHttpEntrypoint(deployment.Config(), 80, domain.HttpEntrypointOptions{})
		udp := appService.AddUDPEntrypoint(8080)
		dbService := deployment.Config().NewService("db", "postgres:14-alpine")
		tcp := dbService.AddTCPEntrypoint(5432)

		services := domain.Services{appService, dbService}

		target.ExposeEntrypoints(deployment.ID().AppID(), deployment.Config().Environment(), services)

		assert.HasNEvents(t, 3, &target)
		evt := assert.EventIs[domain.TargetEntrypointsChanged](t, &target, 1)
		assert.DeepEqual(t, domain.TargetEntrypoints{
			deployment.ID().AppID(): {
				deployment.Config().Environment(): {
					http.Name(): monad.None[domain.Port](),
					udp.Name():  monad.None[domain.Port](),
					tcp.Name():  monad.None[domain.Port](),
				},
			},
		}, evt.Entrypoints)

		changed := assert.EventIs[domain.TargetStateChanged](t, &target, 2)
		assert.Equal(t, domain.TargetStatusConfiguring, changed.State.Status())

		// Should not trigger it again
		target.ExposeEntrypoints(deployment.ID().AppID(), deployment.Config().Environment(), services)
		assert.HasNEvents(t, 3, &target)
	})

	t.Run("should switch to the configuring state if adding new entrypoints to an already exposed environment", func(t *testing.T) {
		target := fixture.Target()
		deployment := fixture.Deployment()
		appService := deployment.Config().NewService("app", "")
		http := appService.AddHttpEntrypoint(deployment.Config(), 80, domain.HttpEntrypointOptions{})

		target.ExposeEntrypoints(deployment.ID().AppID(), deployment.Config().Environment(), domain.Services{appService})

		assert.HasNEvents(t, 3, &target)
		evt := assert.EventIs[domain.TargetEntrypointsChanged](t, &target, 1)
		assert.DeepEqual(t, domain.TargetEntrypoints{
			deployment.ID().AppID(): {
				deployment.Config().Environment(): {
					http.Name(): monad.None[domain.Port](),
				},
			},
		}, evt.Entrypoints)

		// Adding a new entrypoint should trigger new events
		dbService := deployment.Config().NewService("db", "postgres:14-alpine")
		tcp := dbService.AddTCPEntrypoint(5432)

		target.ExposeEntrypoints(deployment.ID().AppID(), deployment.Config().Environment(), domain.Services{appService, dbService})

		assert.HasNEvents(t, 5, &target)
		evt = assert.EventIs[domain.TargetEntrypointsChanged](t, &target, 3)
		assert.DeepEqual(t, domain.TargetEntrypoints{
			deployment.ID().AppID(): {
				deployment.Config().Environment(): {
					http.Name(): monad.None[domain.Port](),
					tcp.Name():  monad.None[domain.Port](),
				},
			},
		}, evt.Entrypoints)

		// Again with the same entrypoints, should trigger nothing new
		target.ExposeEntrypoints(deployment.ID().AppID(), deployment.Config().Environment(), domain.Services{appService, dbService, deployment.Config().NewService("cache", "redis:6-alpine")})
		assert.HasNEvents(t, 5, &target)
	})

	t.Run("should switch to the configuring state if removing entrypoints", func(t *testing.T) {
		target := fixture.Target()
		deployment := fixture.Deployment()
		appService := deployment.Config().NewService("app", "")
		http := appService.AddHttpEntrypoint(deployment.Config(), 80, domain.HttpEntrypointOptions{})
		appService.AddUDPEntrypoint(8080)
		dbService := deployment.Config().NewService("db", "postgres:14-alpine")
		tcp := dbService.AddTCPEntrypoint(5432)

		target.ExposeEntrypoints(deployment.ID().AppID(), deployment.Config().Environment(), domain.Services{appService, dbService})

		// Let's remove the UDP entrypoint
		appService = deployment.Config().NewService("app", "")
		appService.AddHttpEntrypoint(deployment.Config(), 80, domain.HttpEntrypointOptions{})

		target.ExposeEntrypoints(deployment.ID().AppID(), deployment.Config().Environment(), domain.Services{appService, dbService})

		assert.HasNEvents(t, 5, &target)
		evt := assert.EventIs[domain.TargetEntrypointsChanged](t, &target, 3)
		assert.DeepEqual(t, domain.TargetEntrypoints{
			deployment.ID().AppID(): {
				deployment.Config().Environment(): {
					http.Name(): monad.None[domain.Port](),
					tcp.Name():  monad.None[domain.Port](),
				},
			},
		}, evt.Entrypoints)
	})

	t.Run("should remove empty map keys when updating entrypoints", func(t *testing.T) {
		target := fixture.Target()
		deployment := fixture.Deployment()
		appService := deployment.Config().NewService("app", "")
		http := appService.AddHttpEntrypoint(deployment.Config(), 80, domain.HttpEntrypointOptions{})
		tcp := appService.AddTCPEntrypoint(5432)

		target.ExposeEntrypoints(deployment.ID().AppID(), deployment.Config().Environment(), domain.Services{appService})
		assert.DeepEqual(t, domain.TargetEntrypoints{
			deployment.ID().AppID(): {
				domain.Production: {
					http.Name(): monad.None[domain.Port](),
					tcp.Name():  monad.None[domain.Port](),
				},
			},
		}, target.CustomEntrypoints())

		target.ExposeEntrypoints(deployment.ID().AppID(), deployment.Config().Environment(), domain.Services{})

		assert.DeepEqual(t, domain.TargetEntrypoints{}, target.CustomEntrypoints())
	})

	t.Run("should not be removed if no cleanup request has been set", func(t *testing.T) {
		target := fixture.Target()

		err := target.Delete(true)

		assert.ErrorIs(t, domain.ErrTargetCleanupNeeded, err)
	})

	t.Run("should not be removed if target resources have not been cleaned up", func(t *testing.T) {
		target := fixture.Target()
		target.Configured(target.CurrentVersion(), nil, nil)
		assert.Nil(t, target.RequestCleanup(false, "uid")) // No application is using it

		err := target.Delete(false)

		assert.ErrorIs(t, domain.ErrTargetCleanupNeeded, err)
	})

	t.Run("could be removed if resources have been cleaned up", func(t *testing.T) {
		target := fixture.Target()
		target.Configured(target.CurrentVersion(), nil, nil)
		assert.Nil(t, target.RequestCleanup(false, "uid"))

		err := target.Delete(true)

		assert.Nil(t, err)
		assert.EventIs[domain.TargetDeleted](t, &target, 3)
	})
}

func Test_TargetEvents(t *testing.T) {
	t.Run("TargetStateChanged should provide a function to check for configuration changes", func(t *testing.T) {
		target := fixture.Target()
		target.Configured(target.CurrentVersion(), nil, nil)

		evt := assert.EventIs[domain.TargetStateChanged](t, &target, 1)
		assert.False(t, evt.WentToConfiguringState())

		assert.Nil(t, target.Reconfigure())

		evt = assert.EventIs[domain.TargetStateChanged](t, &target, 2)
		assert.True(t, evt.WentToConfiguringState())
	})
}
