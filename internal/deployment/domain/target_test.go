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
	// Common data used for custom entrypoints exposure
	deployment := fixture.Deployment()
	builder := deployment.Config().ServicesBuilder()
	app := builder.AddService("app", "app-image")
	app.AddHttpEntrypoint(80, false)
	http := app.AddHttpEntrypoint(3000, true)
	db := builder.AddService("db", "db-image")
	tcp := db.AddTCPEntrypoint(5432, true)

	t.Run("could be created", func(t *testing.T) {
		t.Run("should require a unique provider config", func(t *testing.T) {
			_, err := domain.NewTarget("target",
				domain.NewProviderConfigRequirement(fixture.ProviderConfig(), false), "uid")

			assert.ErrorIs(t, domain.ErrConfigAlreadyTaken, err)
		})

		t.Run("should succeed if everything is good", func(t *testing.T) {
			config := fixture.ProviderConfig()

			target, err := domain.NewTarget("target",
				domain.NewProviderConfigRequirement(config, true),
				"uid")

			assert.Nil(t, err)
			assert.Equal(t, config, target.Provider())
			assert.Zero(t, target.Url())
			assert.HasNEvents(t, 1, &target)
			created := assert.EventIs[domain.TargetCreated](t, &target, 0)

			assert.DeepEqual(t, domain.TargetCreated{
				ID:          assert.NotZero(t, target.ID()),
				Name:        "target",
				Provider:    config,
				State:       created.State,
				Entrypoints: make(domain.TargetEntrypoints),
				Created:     shared.ActionFrom[auth.UserID]("uid", assert.NotZero(t, created.Created.At())),
			}, created)

			assert.Equal(t, domain.TargetStatusConfiguring, created.State.Status())
			assert.NotZero(t, created.State.Version())
		})
	})

	t.Run("should expose a method to check if a version is outdated or not", func(t *testing.T) {
		t.Run("should return true if the version is outdated", func(t *testing.T) {
			target := fixture.Target()

			assert.True(t, target.IsOutdated(target.CurrentVersion().Add(-1*time.Second)))
		})

		t.Run("should return false if the version is not outdated", func(t *testing.T) {
			target := fixture.Target()

			assert.False(t, target.IsOutdated(target.CurrentVersion()))
		})
	})

	t.Run("could be renamed", func(t *testing.T) {
		t.Run("should not raise the event if the name has not changed", func(t *testing.T) {
			target := fixture.Target(fixture.WithTargetName("name"))

			assert.Nil(t, target.Rename("name"))
			assert.HasNEvents(t, 1, &target)
		})

		t.Run("should raise the event if the name is different", func(t *testing.T) {
			target := fixture.Target(fixture.WithTargetName("old-name"))

			assert.Nil(t, target.Rename("new-name"))
			assert.HasNEvents(t, 2, &target)
			renamed := assert.EventIs[domain.TargetRenamed](t, &target, 1)
			assert.Equal(t, domain.TargetRenamed{
				ID:   target.ID(),
				Name: "new-name",
			}, renamed)
		})

		t.Run("should returns an error if the target cleanup has been requested", func(t *testing.T) {
			target := fixture.Target(fixture.WithTargetName("old-name"))
			target.Configured(target.CurrentVersion(), nil, nil)
			assert.Nil(t, target.RequestCleanup(false, "uid"))

			assert.ErrorIs(t, domain.ErrTargetCleanupRequested, target.Rename("new-name"))
		})
	})

	t.Run("could be configured as exposing services automatically with an url", func(t *testing.T) {
		t.Run("should require the url to be unique", func(t *testing.T) {
			target := fixture.Target()

			assert.ErrorIs(t, domain.ErrUrlAlreadyTaken, target.ExposeServicesAutomatically(
				domain.NewTargetUrlRequirement(must.Panic(domain.UrlFrom("http://example.com")), false),
			))
		})

		t.Run("should raise the event if the url is different", func(t *testing.T) {
			target := fixture.Target()
			url := must.Panic(domain.UrlFrom("http://example.com"))

			assert.Nil(t, target.ExposeServicesAutomatically(domain.NewTargetUrlRequirement(url, true)))

			assert.HasNEvents(t, 3, &target)
			urlChanged := assert.EventIs[domain.TargetUrlChanged](t, &target, 1)
			assert.Equal(t, domain.TargetUrlChanged{
				ID:  target.ID(),
				Url: url,
			}, urlChanged)
			stateChanged := assert.EventIs[domain.TargetStateChanged](t, &target, 2)
			assert.Equal(t, domain.TargetStatusConfiguring, stateChanged.State.Status())
		})

		t.Run("should not raise the event if the url has not changed", func(t *testing.T) {
			target := fixture.Target()
			url := must.Panic(domain.UrlFrom("http://example.com"))
			assert.Nil(t, target.ExposeServicesAutomatically(domain.NewTargetUrlRequirement(url, true)))

			assert.Nil(t, target.ExposeServicesAutomatically(domain.NewTargetUrlRequirement(url, true)))
			assert.HasNEvents(t, 3, &target)
		})

		t.Run("should returns an error if the target cleanup has been requested", func(t *testing.T) {
			target := fixture.Target()
			target.Configured(target.CurrentVersion(), nil, nil)
			assert.Nil(t, target.RequestCleanup(false, "uid"))

			assert.ErrorIs(t, domain.ErrTargetCleanupRequested, target.ExposeServicesAutomatically(
				domain.NewTargetUrlRequirement(must.Panic(domain.UrlFrom("http://example.com")), true),
			))
		})
	})

	t.Run("could be configured as exposing services manually without url", func(t *testing.T) {
		t.Run("should raise the event if the target had previously an url", func(t *testing.T) {
			target := fixture.Target()
			assert.Nil(t, target.ExposeServicesAutomatically(domain.NewTargetUrlRequirement(must.Panic(domain.UrlFrom("http://example.com")), true)))

			assert.Nil(t, target.ExposeServicesManually())

			assert.HasNEvents(t, 5, &target)
			urlRemoved := assert.EventIs[domain.TargetUrlRemoved](t, &target, 3)
			assert.Equal(t, domain.TargetUrlRemoved{
				ID: target.ID(),
			}, urlRemoved)
			stateChanged := assert.EventIs[domain.TargetStateChanged](t, &target, 4)
			assert.Equal(t, domain.TargetStatusConfiguring, stateChanged.State.Status())
		})

		t.Run("should not raise the event if trying to remove an url on a target without one", func(t *testing.T) {
			target := fixture.Target()

			assert.Nil(t, target.ExposeServicesManually())
			assert.HasNEvents(t, 1, &target)
		})

		t.Run("should returns an error if the target cleanup has been requested", func(t *testing.T) {
			target := fixture.Target()
			target.Configured(target.CurrentVersion(), nil, nil)
			assert.Nil(t, target.RequestCleanup(false, "uid"))

			assert.ErrorIs(t, domain.ErrTargetCleanupRequested, target.ExposeServicesManually())
		})
	})

	t.Run("could have its provider changed", func(t *testing.T) {
		t.Run("should require the provider to be unique", func(t *testing.T) {
			target := fixture.Target()

			assert.ErrorIs(t, domain.ErrConfigAlreadyTaken,
				target.HasProvider(domain.NewProviderConfigRequirement(fixture.ProviderConfig(), false)))
		})

		t.Run("should require the fingerprint to be the same", func(t *testing.T) {
			config := fixture.ProviderConfig(fixture.WithKind("test"), fixture.WithFingerprint("123"))
			target := fixture.Target(fixture.WithProviderConfig(config))

			assert.ErrorIs(t, domain.ErrTargetProviderUpdateNotPermitted,
				target.HasProvider(
					domain.NewProviderConfigRequirement(
						fixture.ProviderConfig(fixture.WithKind("test"), fixture.WithFingerprint("456")), true)))
		})

		t.Run("should require the provider kind to be the same", func(t *testing.T) {
			config := fixture.ProviderConfig(fixture.WithKind("test1"), fixture.WithFingerprint("123"))
			target := fixture.Target(fixture.WithProviderConfig(config))

			assert.ErrorIs(t, domain.ErrTargetProviderUpdateNotPermitted,
				target.HasProvider(
					domain.NewProviderConfigRequirement(
						fixture.ProviderConfig(fixture.WithKind("test2"), fixture.WithFingerprint("123")), true)))
		})

		t.Run("should raise the event if the provider is different", func(t *testing.T) {
			config := fixture.ProviderConfig(fixture.WithKind("test"), fixture.WithFingerprint("123"))
			target := fixture.Target(fixture.WithProviderConfig(config))
			newConfig := fixture.ProviderConfig(
				fixture.WithKind("test"),
				fixture.WithFingerprint("123"),
				fixture.WithData("some different data"))

			assert.Nil(t, target.HasProvider(
				domain.NewProviderConfigRequirement(newConfig, true)))
			assert.HasNEvents(t, 3, &target)
			changed := assert.EventIs[domain.TargetProviderChanged](t, &target, 1)
			assert.Equal(t, domain.TargetProviderChanged{
				ID:       target.ID(),
				Provider: newConfig,
			}, changed)
			stateChanged := assert.EventIs[domain.TargetStateChanged](t, &target, 2)
			assert.Equal(t, domain.TargetStatusConfiguring, stateChanged.State.Status())
		})

		t.Run("should not raise the event if the provider is the same", func(t *testing.T) {
			config := fixture.ProviderConfig(fixture.WithKind("test"), fixture.WithFingerprint("123"))
			target := fixture.Target(fixture.WithProviderConfig(config))

			assert.Nil(t, target.HasProvider(domain.NewProviderConfigRequirement(config, true)))

			assert.HasNEvents(t, 1, &target)
		})

		t.Run("should returns an error if the target cleanup has been requested", func(t *testing.T) {
			target := fixture.Target()
			target.Configured(target.CurrentVersion(), nil, nil)
			assert.Nil(t, target.RequestCleanup(false, "uid"))

			assert.ErrorIs(t, domain.ErrTargetCleanupRequested, target.HasProvider(domain.NewProviderConfigRequirement(fixture.ProviderConfig(), true)))
		})
	})

	t.Run("could expose custom entrypoints", func(t *testing.T) {
		t.Run("should do nothing if given entrypoints are empty", func(t *testing.T) {
			target := fixture.Target()

			target.ExposeEntrypoints(deployment.Config().AppID(), domain.Production, domain.Services{})

			assert.HasNEvents(t, 1, &target)
		})

		t.Run("should do nothing if given entrypoints are nil", func(t *testing.T) {
			target := fixture.Target()

			target.ExposeEntrypoints(deployment.Config().AppID(), domain.Production, nil)

			assert.HasNEvents(t, 1, &target)
		})

		t.Run("should add entrypoints", func(t *testing.T) {
			t.Run("on manual target", func(t *testing.T) {
				target := fixture.Target()

				target.ExposeEntrypoints(deployment.Config().AppID(), domain.Production, domain.Services{app.Service, db.Service})

				assert.HasNEvents(t, 2, &target)
				changed := assert.EventIs[domain.TargetEntrypointsChanged](t, &target, 1)
				assert.DeepEqual(t, domain.TargetEntrypointsChanged{
					ID: target.ID(),
					Entrypoints: domain.TargetEntrypoints{
						deployment.Config().AppID(): {
							domain.Production: {
								http.Name(): monad.None[domain.Port](),
								tcp.Name():  monad.None[domain.Port](),
							},
						},
					},
				}, changed)
			})

			t.Run("on automatic target", func(t *testing.T) {
				target := fixture.Target()
				assert.Nil(t, target.ExposeServicesAutomatically(domain.NewTargetUrlRequirement(must.Panic(domain.UrlFrom("http://example.com")), true)))

				target.ExposeEntrypoints(deployment.Config().AppID(), domain.Production, domain.Services{app.Service, db.Service})

				assert.HasNEvents(t, 5, &target)
				changed := assert.EventIs[domain.TargetEntrypointsChanged](t, &target, 3)
				assert.DeepEqual(t, domain.TargetEntrypointsChanged{
					ID: target.ID(),
					Entrypoints: domain.TargetEntrypoints{
						deployment.Config().AppID(): {
							domain.Production: {
								http.Name(): monad.None[domain.Port](),
								tcp.Name():  monad.None[domain.Port](),
							},
						},
					},
				}, changed)
				stateChanged := assert.EventIs[domain.TargetStateChanged](t, &target, 4)
				assert.Equal(t, domain.TargetStatusConfiguring, stateChanged.State.Status())
			})
		})

		t.Run("should update existing entrypoints", func(t *testing.T) {
			t.Run("on manual target", func(t *testing.T) {
				target := fixture.Target()
				target.ExposeEntrypoints(deployment.Config().AppID(), domain.Production, domain.Services{app.Service, db.Service})

				target.ExposeEntrypoints(deployment.Config().AppID(), domain.Production, domain.Services{app.Service})

				assert.HasNEvents(t, 3, &target)
				changed := assert.EventIs[domain.TargetEntrypointsChanged](t, &target, 2)
				assert.DeepEqual(t, domain.TargetEntrypointsChanged{
					ID: target.ID(),
					Entrypoints: domain.TargetEntrypoints{
						deployment.Config().AppID(): {
							domain.Production: {
								http.Name(): monad.None[domain.Port](),
							},
						},
					},
				}, changed)
			})

			t.Run("on automatic target", func(t *testing.T) {
				target := fixture.Target()
				assert.Nil(t, target.ExposeServicesAutomatically(domain.NewTargetUrlRequirement(must.Panic(domain.UrlFrom("http://example.com")), true)))
				target.ExposeEntrypoints(deployment.Config().AppID(), domain.Production, domain.Services{app.Service, db.Service})

				target.ExposeEntrypoints(deployment.Config().AppID(), domain.Production, domain.Services{app.Service})

				assert.HasNEvents(t, 7, &target)
				changed := assert.EventIs[domain.TargetEntrypointsChanged](t, &target, 5)
				assert.DeepEqual(t, domain.TargetEntrypointsChanged{
					ID: target.ID(),
					Entrypoints: domain.TargetEntrypoints{
						deployment.Config().AppID(): {
							domain.Production: {
								http.Name(): monad.None[domain.Port](),
							},
						},
					},
				}, changed)
				stateChanged := assert.EventIs[domain.TargetStateChanged](t, &target, 6)
				assert.Equal(t, domain.TargetStatusConfiguring, stateChanged.State.Status())
			})
		})

		t.Run("should not raise additional events if all entrypoints already exists", func(t *testing.T) {
			target := fixture.Target()
			target.ExposeEntrypoints(deployment.Config().AppID(), domain.Production, domain.Services{app.Service, db.Service})

			target.ExposeEntrypoints(deployment.Config().AppID(), domain.Production, domain.Services{app.Service, db.Service})
			assert.HasNEvents(t, 2, &target)
		})

		t.Run("should be ignored if the target is being configured", func(t *testing.T) {
			target := fixture.Target()
			target.Configured(target.CurrentVersion(), nil, nil)
			assert.Nil(t, target.RequestCleanup(false, "uid"))

			target.ExposeEntrypoints(deployment.Config().AppID(), domain.Production, domain.Services{app.Service, db.Service})

			assert.HasNEvents(t, 3, &target)
		})
	})

	t.Run("could be marked as configured", func(t *testing.T) {
		t.Run("should do nothing if the version do not match", func(t *testing.T) {
			target := fixture.Target()
			target.ExposeEntrypoints(deployment.Config().AppID(), domain.Production, domain.Services{app.Service, db.Service})

			target.Configured(target.CurrentVersion().Add(-1*time.Second), domain.TargetEntrypointsAssigned{
				deployment.Config().AppID(): {
					domain.Production: {
						http.Name(): 3000,
						tcp.Name():  3001,
					},
				},
			}, nil)

			assert.HasNEvents(t, 2, &target)
		})

		t.Run("should do nothing if the version has already been configured", func(t *testing.T) {
			target := fixture.Target()
			target.Configured(target.CurrentVersion(), nil, nil)

			target.Configured(target.CurrentVersion(), nil, nil)

			assert.HasNEvents(t, 2, &target)
			stateChanged := assert.EventIs[domain.TargetStateChanged](t, &target, 1)
			assert.Equal(t, domain.TargetStatusReady, stateChanged.State.Status())
		})

		t.Run("should be marked as failed if an error is given", func(t *testing.T) {
			target := fixture.Target()
			target.ExposeEntrypoints(deployment.Config().AppID(), domain.Production, domain.Services{app.Service, db.Service})
			err := errors.New("an error")

			target.Configured(target.CurrentVersion(), domain.TargetEntrypointsAssigned{
				deployment.Config().AppID(): {
					domain.Production: {
						http.Name(): 3000,
						tcp.Name():  3001,
					},
				},
			}, err)

			assert.HasNEvents(t, 3, &target)
			stateChanged := assert.EventIs[domain.TargetStateChanged](t, &target, 2)
			assert.Equal(t, domain.TargetStatusFailed, stateChanged.State.Status())
			assert.Equal(t, err.Error(), stateChanged.State.ErrCode().Get(""))
		})

		t.Run("should be marked as ready and update entrypoints with given assigned ports ignoring non-existing entrypoints", func(t *testing.T) {
			target := fixture.Target()
			target.ExposeEntrypoints(deployment.Config().AppID(), domain.Production, domain.Services{app.Service, db.Service})
			target.ExposeEntrypoints(deployment.Config().AppID(), domain.Staging, domain.Services{app.Service, db.Service})

			target.Configured(target.CurrentVersion(), domain.TargetEntrypointsAssigned{
				"another-app": {
					domain.Production: {
						"some-entrypoint": 5000,
					},
				},
				deployment.Config().AppID(): {
					domain.Production: {
						http.Name(): 3000,
						tcp.Name():  3001,
					},
				},
			}, nil)

			assert.HasNEvents(t, 5, &target)
			entrypointsChanged := assert.EventIs[domain.TargetEntrypointsChanged](t, &target, 3)
			assert.DeepEqual(t, domain.TargetEntrypointsChanged{
				ID: target.ID(),
				Entrypoints: domain.TargetEntrypoints{
					deployment.Config().AppID(): {
						domain.Staging: {
							http.Name(): monad.None[domain.Port](),
							tcp.Name():  monad.None[domain.Port](),
						},
						domain.Production: {
							http.Name(): monad.Value[domain.Port](3000),
							tcp.Name():  monad.Value[domain.Port](3001),
						},
					},
				},
			}, entrypointsChanged)
			stateChanged := assert.EventIs[domain.TargetStateChanged](t, &target, 4)
			assert.Equal(t, domain.TargetStatusReady, stateChanged.State.Status())
		})
	})

	t.Run("could un-expose custom entrypoints", func(t *testing.T) {
		t.Run("should do nothing if not previously exposed", func(t *testing.T) {
			target := fixture.Target()

			target.UnExposeEntrypoints(deployment.Config().AppID(), domain.Production)

			assert.HasNEvents(t, 1, &target)
		})

		t.Run("should un-expose all entrypoints of a given application", func(t *testing.T) {
			target := fixture.Target()
			target.ExposeEntrypoints("app", domain.Production, domain.Services{app.Service, db.Service})
			target.ExposeEntrypoints(deployment.Config().AppID(), domain.Production, domain.Services{app.Service, db.Service})
			target.ExposeEntrypoints(deployment.Config().AppID(), domain.Staging, domain.Services{app.Service, db.Service})
			target.Configured(target.CurrentVersion(), domain.TargetEntrypointsAssigned{
				deployment.Config().AppID(): {
					domain.Production: {
						http.Name(): 3000,
						tcp.Name():  3001,
					},
					domain.Staging: {
						http.Name(): 3002,
						tcp.Name():  3003,
					},
				},
			}, nil)

			target.UnExposeEntrypoints(deployment.Config().AppID())

			assert.HasNEvents(t, 7, &target)
			entrypointsChanged := assert.EventIs[domain.TargetEntrypointsChanged](t, &target, 6)
			assert.DeepEqual(t, domain.TargetEntrypointsChanged{
				ID: target.ID(),
				Entrypoints: domain.TargetEntrypoints{
					"app": {
						domain.Production: {
							http.Name(): monad.None[domain.Port](),
							tcp.Name():  monad.None[domain.Port](),
						},
					},
				},
			}, entrypointsChanged)
		})

		t.Run("should un-expose all entrypoints of an application for a specific environment", func(t *testing.T) {
			t.Run("on manual target", func(t *testing.T) {
				target := fixture.Target()
				target.ExposeEntrypoints(deployment.Config().AppID(), domain.Production, domain.Services{app.Service, db.Service})
				target.ExposeEntrypoints(deployment.Config().AppID(), domain.Staging, domain.Services{app.Service, db.Service})
				target.Configured(target.CurrentVersion(), domain.TargetEntrypointsAssigned{
					deployment.Config().AppID(): {
						domain.Production: {
							http.Name(): 3000,
							tcp.Name():  3001,
						},
						domain.Staging: {
							http.Name(): 3002,
							tcp.Name():  3003,
						},
					},
				}, nil)

				target.UnExposeEntrypoints(deployment.Config().AppID(), domain.Production)

				assert.HasNEvents(t, 6, &target)
				entrypointsChanged := assert.EventIs[domain.TargetEntrypointsChanged](t, &target, 5)
				assert.DeepEqual(t, domain.TargetEntrypointsChanged{
					ID: target.ID(),
					Entrypoints: domain.TargetEntrypoints{
						deployment.Config().AppID(): {
							domain.Staging: {
								http.Name(): monad.Value[domain.Port](3002),
								tcp.Name():  monad.Value[domain.Port](3003),
							},
						},
					},
				}, entrypointsChanged)
			})

			t.Run("on automatic target", func(t *testing.T) {
				target := fixture.Target()
				assert.Nil(t, target.ExposeServicesAutomatically(domain.NewTargetUrlRequirement(must.Panic(domain.UrlFrom("https://example.com")), true)))
				target.ExposeEntrypoints(deployment.Config().AppID(), domain.Production, domain.Services{app.Service, db.Service})
				target.ExposeEntrypoints(deployment.Config().AppID(), domain.Staging, domain.Services{app.Service, db.Service})
				target.Configured(target.CurrentVersion(), domain.TargetEntrypointsAssigned{
					deployment.Config().AppID(): {
						domain.Production: {
							http.Name(): 3000,
							tcp.Name():  3001,
						},
						domain.Staging: {
							http.Name(): 3002,
							tcp.Name():  3003,
						},
					},
				}, nil)

				target.UnExposeEntrypoints(deployment.Config().AppID(), domain.Production)

				assert.HasNEvents(t, 11, &target)
				entrypointsChanged := assert.EventIs[domain.TargetEntrypointsChanged](t, &target, 9)
				assert.DeepEqual(t, domain.TargetEntrypointsChanged{
					ID: target.ID(),
					Entrypoints: domain.TargetEntrypoints{
						deployment.Config().AppID(): {
							domain.Staging: {
								http.Name(): monad.Value[domain.Port](3002),
								tcp.Name():  monad.Value[domain.Port](3003),
							},
						},
					},
				}, entrypointsChanged)
				stateChanged := assert.EventIs[domain.TargetStateChanged](t, &target, 10)
				assert.Equal(t, domain.TargetStatusConfiguring, stateChanged.State.Status())
			})
		})

		t.Run("should be ignored if the target cleanup has been requested", func(t *testing.T) {
			target := fixture.Target()
			target.ExposeEntrypoints(deployment.Config().AppID(), domain.Production, domain.Services{app.Service, db.Service})
			target.Configured(target.CurrentVersion(), domain.TargetEntrypointsAssigned{
				deployment.Config().AppID(): {
					domain.Production: {
						http.Name(): 3000,
						tcp.Name():  3001,
					},
				},
			}, nil)
			assert.Nil(t, target.RequestCleanup(false, "uid"))

			target.UnExposeEntrypoints(deployment.Config().AppID(), domain.Production)

			assert.HasNEvents(t, 5, &target)
		})
	})

	t.Run("could expose its availability based on its internal state", func(t *testing.T) {
		t.Run("when configuring", func(t *testing.T) {
			target := fixture.Target()

			assert.ErrorIs(t, domain.ErrTargetConfigurationInProgress, target.CheckAvailability())
		})

		t.Run("when configuration failed", func(t *testing.T) {
			target := fixture.Target()
			target.Configured(target.CurrentVersion(), nil, errors.New("configuration failed"))

			assert.ErrorIs(t, domain.ErrTargetConfigurationFailed, target.CheckAvailability())
		})

		t.Run("when ready", func(t *testing.T) {
			target := fixture.Target()
			target.Configured(target.CurrentVersion(), nil, nil)

			assert.Nil(t, target.CheckAvailability())
		})

		t.Run("when cleanup requested", func(t *testing.T) {
			target := fixture.Target()
			target.Configured(target.CurrentVersion(), nil, nil)
			assert.Nil(t, target.RequestCleanup(false, "uid"))

			assert.ErrorIs(t, domain.ErrTargetCleanupRequested, target.CheckAvailability())
		})
	})

	t.Run("could be reconfigured", func(t *testing.T) {
		t.Run("should fail if already being configured", func(t *testing.T) {
			target := fixture.Target()

			assert.ErrorIs(t, domain.ErrTargetConfigurationInProgress, target.Reconfigure())
		})

		t.Run("should fail if cleanup requested", func(t *testing.T) {
			target := fixture.Target()
			target.Configured(target.CurrentVersion(), nil, nil)
			assert.Nil(t, target.RequestCleanup(false, "uid"))

			assert.ErrorIs(t, domain.ErrTargetCleanupRequested, target.Reconfigure())
		})

		t.Run("should succeed otherwise", func(t *testing.T) {
			target := fixture.Target()
			target.Configured(target.CurrentVersion(), nil, nil)

			assert.Nil(t, target.Reconfigure())

			assert.HasNEvents(t, 3, &target)
			stateChanged := assert.EventIs[domain.TargetStateChanged](t, &target, 2)
			assert.Equal(t, domain.TargetStatusConfiguring, stateChanged.State.Status())
		})
	})

	t.Run("could be marked for cleanup", func(t *testing.T) {
		t.Run("should returns an err if some applications are using it", func(t *testing.T) {
			target := fixture.Target()

			assert.ErrorIs(t, domain.ErrTargetInUse, target.RequestCleanup(true, "uid"))
		})

		t.Run("should returns an err if configuring", func(t *testing.T) {
			target := fixture.Target()

			assert.ErrorIs(t, domain.ErrTargetConfigurationInProgress, target.RequestCleanup(false, "uid"))
		})

		t.Run("should succeed otherwise", func(t *testing.T) {
			target := fixture.Target()
			target.Configured(target.CurrentVersion(), nil, nil)

			assert.Nil(t, target.RequestCleanup(false, "uid"))

			assert.HasNEvents(t, 3, &target)
			requested := assert.EventIs[domain.TargetCleanupRequested](t, &target, 2)
			assert.Equal(t, domain.TargetCleanupRequested{
				ID:        target.ID(),
				Requested: shared.ActionFrom[auth.UserID]("uid", assert.NotZero(t, requested.Requested.At())),
			}, requested)
		})

		t.Run("should do nothing if already being cleaned up", func(t *testing.T) {
			target := fixture.Target()
			target.Configured(target.CurrentVersion(), nil, nil)
			assert.Nil(t, target.RequestCleanup(false, "uid"))

			assert.Nil(t, target.RequestCleanup(false, "uid"))

			assert.HasNEvents(t, 3, &target)
		})
	})

	t.Run("should expose a cleanup strategy to determine how the target resources should be handled", func(t *testing.T) {
		t.Run("should returns an error if there are running or pending deployments on the target", func(t *testing.T) {
			target := fixture.Target()

			_, err := target.CleanupStrategy(true)

			assert.ErrorIs(t, domain.ErrRunningOrPendingDeployments, err)
		})

		t.Run("should returns an error if the target is being configured", func(t *testing.T) {
			target := fixture.Target()

			_, err := target.CleanupStrategy(false)

			assert.ErrorIs(t, domain.ErrTargetConfigurationInProgress, err)
		})

		t.Run("should returns an error if the target configuration has failed and it has been at least ready once", func(t *testing.T) {
			target := fixture.Target()
			target.Configured(target.CurrentVersion(), nil, nil)
			assert.Nil(t, target.Reconfigure())
			target.Configured(target.CurrentVersion(), nil, errors.New("failed"))

			_, err := target.CleanupStrategy(false)

			assert.ErrorIs(t, domain.ErrTargetConfigurationFailed, err)
		})

		t.Run("should returns the skip strategy if the target has never been correctly configured and is currently failing", func(t *testing.T) {
			target := fixture.Target()
			target.Configured(target.CurrentVersion(), nil, errors.New("failed"))

			strategy, err := target.CleanupStrategy(false)

			assert.Nil(t, err)
			assert.Equal(t, domain.CleanupStrategySkip, strategy)
		})

		t.Run("should returns the default strategy if the target is ready", func(t *testing.T) {
			target := fixture.Target()
			target.Configured(target.CurrentVersion(), nil, nil)

			strategy, err := target.CleanupStrategy(false)

			assert.Nil(t, err)
			assert.Equal(t, domain.CleanupStrategyDefault, strategy)
		})
	})

	t.Run("should expose an application cleanup strategy to determine how application resources should be handled", func(t *testing.T) {
		t.Run("should returns the skip strategy if the target is being cleaned up", func(t *testing.T) {
			target := fixture.Target()
			target.Configured(target.CurrentVersion(), nil, nil)
			assert.Nil(t, target.RequestCleanup(false, "uid"))

			strategy, err := target.AppCleanupStrategy(false, true)

			assert.Nil(t, err)
			assert.Equal(t, domain.CleanupStrategySkip, strategy)
		})

		t.Run("should returns an error if there are still running deployments on the target for this application", func(t *testing.T) {
			target := fixture.Target()

			_, err := target.AppCleanupStrategy(true, true)

			assert.ErrorIs(t, domain.ErrRunningOrPendingDeployments, err)
		})

		t.Run("should returns the skip strategy if no successful deployment has been made and no one is running", func(t *testing.T) {
			target := fixture.Target()

			strategy, err := target.AppCleanupStrategy(false, false)

			assert.Nil(t, err)
			assert.Equal(t, domain.CleanupStrategySkip, strategy)
		})

		t.Run("should returns an error if the target is being configured", func(t *testing.T) {
			target := fixture.Target()

			_, err := target.AppCleanupStrategy(false, true)

			assert.ErrorIs(t, domain.ErrTargetConfigurationInProgress, err)
		})

		t.Run("should returns an error if the target configuration has failed", func(t *testing.T) {
			target := fixture.Target()
			target.Configured(target.CurrentVersion(), nil, errors.New("failed"))

			_, err := target.AppCleanupStrategy(false, true)

			assert.ErrorIs(t, domain.ErrTargetConfigurationFailed, err)
		})

		t.Run("should returns the default strategy if the target is ready", func(t *testing.T) {
			target := fixture.Target()
			target.Configured(target.CurrentVersion(), nil, nil)

			strategy, err := target.AppCleanupStrategy(false, true)

			assert.Nil(t, err)
			assert.Equal(t, domain.CleanupStrategyDefault, strategy)
		})
	})

	t.Run("could be deleted", func(t *testing.T) {
		t.Run("should returns an error if the target has not been mark for cleanup", func(t *testing.T) {
			target := fixture.Target()

			assert.ErrorIs(t, domain.ErrTargetCleanupNeeded, target.Delete(true))
		})

		t.Run("should returns an error if the target resources has not been cleaned up", func(t *testing.T) {
			target := fixture.Target()

			assert.ErrorIs(t, domain.ErrTargetCleanupNeeded, target.Delete(false))
		})

		t.Run("should succeed otherwise", func(t *testing.T) {
			target := fixture.Target()
			target.Configured(target.CurrentVersion(), nil, nil)
			assert.Nil(t, target.RequestCleanup(false, "uid"))

			assert.Nil(t, target.Delete(true))
			assert.HasNEvents(t, 4, &target)
			deleted := assert.EventIs[domain.TargetDeleted](t, &target, 3)
			assert.Equal(t, domain.TargetDeleted{
				ID: target.ID(),
			}, deleted)
		})
	})
}

func Test_TargetEvents(t *testing.T) {
	t.Run("should provide a function to check for configuration changes", func(t *testing.T) {
		t.Run("should return false if the state is not configuring", func(t *testing.T) {
			target := fixture.Target()
			target.Configured(target.CurrentVersion(), nil, nil)

			evt := assert.EventIs[domain.TargetStateChanged](t, &target, 1)
			assert.False(t, evt.WentToConfiguringState())
		})

		t.Run("should return true if going to the configuring state", func(t *testing.T) {
			target := fixture.Target()
			target.Configured(target.CurrentVersion(), nil, nil)
			assert.Nil(t, target.Reconfigure())

			evt := assert.EventIs[domain.TargetStateChanged](t, &target, 2)
			assert.True(t, evt.WentToConfiguringState())
		})
	})
}

func Test_TargetEntrypointsAssigned(t *testing.T) {
	t.Run("should provide a function to set entrypoints values", func(t *testing.T) {
		assigned := make(domain.TargetEntrypointsAssigned)

		assigned.Set("app", domain.Production, "http", 3000)
		assigned.Set("app", domain.Production, "tcp", 3001)
		assigned.Set("app", domain.Staging, "http", 3002)

		assert.DeepEqual(t, domain.TargetEntrypointsAssigned{
			"app": {
				domain.Production: {
					"http": 3000,
					"tcp":  3001,
				},
				domain.Staging: {
					"http": 3002,
				},
			},
		}, assigned)
	})
}
