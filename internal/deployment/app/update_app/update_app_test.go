package update_app_test

import (
	"context"
	"testing"

	authfixture "github.com/YuukanOO/seelf/internal/auth/fixture"
	"github.com/YuukanOO/seelf/internal/deployment/app/update_app"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/deployment/fixture"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/assert"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/bus/spy"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/must"
	"github.com/YuukanOO/seelf/pkg/validate"
)

func Test_UpdateApp(t *testing.T) {

	arrange := func(tb testing.TB, seed ...fixture.SeedBuilder) (
		bus.RequestHandler[string, update_app.Command],
		context.Context,
		spy.Dispatcher,
	) {
		context := fixture.PrepareDatabase(tb, seed...)
		return update_app.Handler(context.AppsStore, context.AppsStore), context.Context, context.Dispatcher
	}

	t.Run("should require a valid application id", func(t *testing.T) {
		handler, ctx, _ := arrange(t)

		id, err := handler(ctx, update_app.Command{})

		assert.ErrorIs(t, apperr.ErrNotFound, err)
		assert.Zero(t, id)
	})

	t.Run("should update nothing if no fields are provided", func(t *testing.T) {
		user := authfixture.User()
		target := fixture.Target(fixture.WithTargetCreatedBy(user.ID()))
		app := fixture.App(
			fixture.WithAppCreatedBy(user.ID()),
			fixture.WithEnvironmentConfig(
				domain.NewEnvironmentConfig(target.ID()),
				domain.NewEnvironmentConfig(target.ID()),
			),
		)
		handler, ctx, dispatcher := arrange(t,
			fixture.WithUsers(&user),
			fixture.WithTargets(&target),
			fixture.WithApps(&app),
		)

		id, err := handler(ctx, update_app.Command{
			ID: string(app.ID()),
		})

		assert.Nil(t, err)
		assert.Equal(t, string(app.ID()), id)
		assert.HasLength(t, 0, dispatcher.Signals())
	})

	t.Run("should validate new target naming availability", func(t *testing.T) {
		user := authfixture.User()
		targetOne := fixture.Target(fixture.WithTargetCreatedBy(user.ID()))
		targetTwo := fixture.Target(fixture.WithTargetCreatedBy(user.ID()))
		appOne := fixture.App(
			fixture.WithAppCreatedBy(user.ID()),
			fixture.WithAppName("my-app"),
			fixture.WithEnvironmentConfig(
				domain.NewEnvironmentConfig(targetOne.ID()),
				domain.NewEnvironmentConfig(targetOne.ID()),
			),
		)
		appTwo := fixture.App(
			fixture.WithAppCreatedBy(user.ID()),
			fixture.WithAppName("my-app"),
			fixture.WithEnvironmentConfig(
				domain.NewEnvironmentConfig(targetTwo.ID()),
				domain.NewEnvironmentConfig(targetTwo.ID()),
			),
		)
		handler, ctx, _ := arrange(t,
			fixture.WithUsers(&user),
			fixture.WithTargets(&targetOne, &targetTwo),
			fixture.WithApps(&appOne, &appTwo),
		)

		_, err := handler(ctx, update_app.Command{
			ID: string(appTwo.ID()),
			Production: monad.Value(update_app.EnvironmentConfig{
				Target: string(targetOne.ID()),
			}),
			Staging: monad.Value(update_app.EnvironmentConfig{
				Target: string(targetOne.ID()),
			}),
		})

		assert.ValidationError(t, validate.FieldErrors{
			"production.target": domain.ErrAppNameAlreadyTaken,
			"staging.target":    domain.ErrAppNameAlreadyTaken,
		}, err)
	})

	t.Run("should remove an application env variables", func(t *testing.T) {
		user := authfixture.User()
		target := fixture.Target(fixture.WithTargetCreatedBy(user.ID()))
		otherTarget := fixture.Target(fixture.WithTargetCreatedBy(user.ID()))
		configWithEnvVariables := domain.NewEnvironmentConfig(target.ID())
		configWithEnvVariables.HasEnvironmentVariables(domain.ServicesEnv{
			"app": {"DEBUG": "false"},
		})
		app := fixture.App(
			fixture.WithAppCreatedBy(user.ID()),
			fixture.WithEnvironmentConfig(
				configWithEnvVariables,
				configWithEnvVariables,
			),
		)
		handler, ctx, dispatcher := arrange(t,
			fixture.WithUsers(&user),
			fixture.WithTargets(&target, &otherTarget),
			fixture.WithApps(&app),
		)

		id, err := handler(ctx, update_app.Command{
			ID: string(app.ID()),
			Production: monad.Value(update_app.EnvironmentConfig{
				Target: string(otherTarget.ID()),
			}),
			Staging: monad.Value(update_app.EnvironmentConfig{
				Target: string(otherTarget.ID()),
			}),
		})

		assert.Nil(t, err)
		assert.Equal(t, string(app.ID()), id)
		assert.HasLength(t, 4, dispatcher.Signals())

		changed := assert.Is[domain.AppEnvChanged](t, dispatcher.Signals()[0])
		assert.Equal(t, domain.Production, changed.Environment)
		assert.Equal(t, otherTarget.ID(), changed.Config.Target())
		assert.False(t, changed.Config.Vars().HasValue())
		historyChanged := assert.Is[domain.AppHistoryChanged](t, dispatcher.Signals()[1])
		assert.DeepEqual(t, domain.AppHistoryChanged{
			ID: app.ID(),
			History: domain.AppTargetHistory{
				domain.Production: []domain.TargetID{
					target.ID(),
					otherTarget.ID(),
				},
				domain.Staging: []domain.TargetID{
					target.ID(),
					otherTarget.ID(),
				},
			},
		}, historyChanged)

		changed = assert.Is[domain.AppEnvChanged](t, dispatcher.Signals()[2])
		assert.Equal(t, domain.Staging, changed.Environment)
		assert.Equal(t, otherTarget.ID(), changed.Config.Target())
		assert.False(t, changed.Config.Vars().HasValue())
		historyChanged = assert.Is[domain.AppHistoryChanged](t, dispatcher.Signals()[3])
		assert.DeepEqual(t, domain.AppHistoryChanged{
			ID: app.ID(),
			History: domain.AppTargetHistory{
				domain.Production: []domain.TargetID{
					target.ID(),
					otherTarget.ID(),
				},
				domain.Staging: []domain.TargetID{
					target.ID(),
					otherTarget.ID(),
				},
			},
		}, historyChanged)
	})

	t.Run("should update an application env variables", func(t *testing.T) {
		user := authfixture.User()
		target := fixture.Target(fixture.WithTargetCreatedBy(user.ID()))
		configWithEnvVariables := domain.NewEnvironmentConfig(target.ID())
		configWithEnvVariables.HasEnvironmentVariables(domain.ServicesEnv{
			"app": {"DEBUG": "false"},
		})
		app := fixture.App(
			fixture.WithAppCreatedBy(user.ID()),
			fixture.WithEnvironmentConfig(
				configWithEnvVariables,
				configWithEnvVariables,
			),
		)
		handler, ctx, dispatcher := arrange(t,
			fixture.WithUsers(&user),
			fixture.WithTargets(&target),
			fixture.WithApps(&app),
		)

		id, err := handler(ctx, update_app.Command{
			ID: string(app.ID()),
			Production: monad.Value(update_app.EnvironmentConfig{
				Target: string(target.ID()),
				Vars: monad.Value(map[string]map[string]string{
					"app": {"OTHER": "value"},
				}),
			}),
			Staging: monad.Value(update_app.EnvironmentConfig{
				Target: string(target.ID()),
				Vars: monad.Value(map[string]map[string]string{
					"app": {"SOMETHING": "else"},
				}),
			}),
		})

		assert.Nil(t, err)
		assert.Equal(t, string(app.ID()), id)
		assert.HasLength(t, 2, dispatcher.Signals())

		changed := assert.Is[domain.AppEnvChanged](t, dispatcher.Signals()[0])
		assert.Equal(t, domain.Production, changed.Environment)
		assert.Equal(t, target.ID(), changed.Config.Target())
		assert.DeepEqual(t, domain.ServicesEnv{
			"app": {"OTHER": "value"},
		}, changed.Config.Vars().MustGet())

		changed = assert.Is[domain.AppEnvChanged](t, dispatcher.Signals()[1])
		assert.Equal(t, domain.Staging, changed.Environment)
		assert.Equal(t, target.ID(), changed.Config.Target())
		assert.DeepEqual(t, domain.ServicesEnv{
			"app": {"SOMETHING": "else"},
		}, changed.Config.Vars().MustGet())
	})

	t.Run("should require valid vcs inputs", func(t *testing.T) {
		user := authfixture.User()
		target := fixture.Target(fixture.WithTargetCreatedBy(user.ID()))
		app := fixture.App(
			fixture.WithAppCreatedBy(user.ID()),
			fixture.WithEnvironmentConfig(
				domain.NewEnvironmentConfig(target.ID()),
				domain.NewEnvironmentConfig(target.ID()),
			),
		)
		handler, ctx, _ := arrange(t,
			fixture.WithUsers(&user),
			fixture.WithTargets(&target),
			fixture.WithApps(&app),
		)

		_, err := handler(ctx, update_app.Command{
			ID: string(app.ID()),
			VersionControl: monad.PatchValue(update_app.VersionControl{
				Url: "invalid-url",
			}),
		})

		assert.ValidationError(t, validate.FieldErrors{
			"version_control.url": domain.ErrInvalidUrl,
		}, err)
	})

	t.Run("should fail if trying to update an app being deleted", func(t *testing.T) {
		user := authfixture.User()
		target := fixture.Target(fixture.WithTargetCreatedBy(user.ID()))
		app := fixture.App(
			fixture.WithAppCreatedBy(user.ID()),
			fixture.WithEnvironmentConfig(
				domain.NewEnvironmentConfig(target.ID()),
				domain.NewEnvironmentConfig(target.ID()),
			),
		)
		assert.Nil(t, app.RequestDelete(user.ID()))
		handler, ctx, _ := arrange(t,
			fixture.WithUsers(&user),
			fixture.WithTargets(&target),
			fixture.WithApps(&app),
		)

		_, err := handler(ctx, update_app.Command{
			ID: string(app.ID()),
			VersionControl: monad.PatchValue(update_app.VersionControl{
				Url: "https://some.url",
			}),
		})

		assert.ErrorIs(t, domain.ErrAppCleanupRequested, err)
	})

	t.Run("should fail if trying to add a vcs config without an url defined", func(t *testing.T) {
		user := authfixture.User()
		target := fixture.Target(fixture.WithTargetCreatedBy(user.ID()))
		app := fixture.App(
			fixture.WithAppCreatedBy(user.ID()),
			fixture.WithEnvironmentConfig(
				domain.NewEnvironmentConfig(target.ID()),
				domain.NewEnvironmentConfig(target.ID()),
			),
		)
		handler, ctx, _ := arrange(t,
			fixture.WithUsers(&user),
			fixture.WithTargets(&target),
			fixture.WithApps(&app),
		)

		_, err := handler(ctx, update_app.Command{
			ID:             string(app.ID()),
			VersionControl: monad.PatchValue(update_app.VersionControl{}),
		})

		assert.ValidationError(t, validate.FieldErrors{
			"version_control.url": domain.ErrInvalidUrl,
		}, err)
	})

	t.Run("should remove the vcs config if nil given", func(t *testing.T) {
		user := authfixture.User()
		target := fixture.Target(fixture.WithTargetCreatedBy(user.ID()))
		app := fixture.App(
			fixture.WithAppCreatedBy(user.ID()),
			fixture.WithEnvironmentConfig(
				domain.NewEnvironmentConfig(target.ID()),
				domain.NewEnvironmentConfig(target.ID()),
			),
		)
		assert.Nil(t, app.UseVersionControl(domain.NewVersionControl(must.Panic(domain.UrlFrom("https://some.url")))))
		handler, ctx, dispatcher := arrange(t,
			fixture.WithUsers(&user),
			fixture.WithTargets(&target),
			fixture.WithApps(&app),
		)

		id, err := handler(ctx, update_app.Command{
			ID:             string(app.ID()),
			VersionControl: monad.Nil[update_app.VersionControl](),
		})

		assert.Nil(t, err)
		assert.Equal(t, string(app.ID()), id)
		assert.HasLength(t, 1, dispatcher.Signals())
		removed := assert.Is[domain.AppVersionControlRemoved](t, dispatcher.Signals()[0])
		assert.Equal(t, domain.AppVersionControlRemoved{
			ID: app.ID(),
		}, removed)
	})

	t.Run("should update the vcs url and keep the token if defined", func(t *testing.T) {
		user := authfixture.User()
		target := fixture.Target(fixture.WithTargetCreatedBy(user.ID()))
		app := fixture.App(
			fixture.WithAppCreatedBy(user.ID()),
			fixture.WithEnvironmentConfig(
				domain.NewEnvironmentConfig(target.ID()),
				domain.NewEnvironmentConfig(target.ID()),
			),
		)
		vcs := domain.NewVersionControl(must.Panic(domain.UrlFrom("https://some.url")))
		vcs.Authenticated("a token")
		assert.Nil(t, app.UseVersionControl(vcs))
		handler, ctx, dispatcher := arrange(t,
			fixture.WithUsers(&user),
			fixture.WithTargets(&target),
			fixture.WithApps(&app),
		)

		id, err := handler(ctx, update_app.Command{
			ID: string(app.ID()),
			VersionControl: monad.PatchValue(update_app.VersionControl{
				Url: "https://some.other.url",
			}),
		})

		assert.Nil(t, err)
		assert.Equal(t, string(app.ID()), id)
		assert.HasLength(t, 1, dispatcher.Signals())
		configured := assert.Is[domain.AppVersionControlConfigured](t, dispatcher.Signals()[0])
		assert.Equal(t, app.ID(), configured.ID)
		assert.Equal(t, "https://some.other.url", configured.Config.Url().String())
		assert.Equal(t, "a token", configured.Config.Token().MustGet())
	})

	t.Run("should remove the vcs token", func(t *testing.T) {
		user := authfixture.User()
		target := fixture.Target(fixture.WithTargetCreatedBy(user.ID()))
		app := fixture.App(
			fixture.WithAppCreatedBy(user.ID()),
			fixture.WithEnvironmentConfig(
				domain.NewEnvironmentConfig(target.ID()),
				domain.NewEnvironmentConfig(target.ID()),
			),
		)
		url := must.Panic(domain.UrlFrom("https://some.url"))
		vcs := domain.NewVersionControl(url)
		vcs.Authenticated("a token")
		assert.Nil(t, app.UseVersionControl(vcs))
		handler, ctx, dispatcher := arrange(t,
			fixture.WithUsers(&user),
			fixture.WithTargets(&target),
			fixture.WithApps(&app),
		)

		id, err := handler(ctx, update_app.Command{
			ID: string(app.ID()),
			VersionControl: monad.PatchValue(update_app.VersionControl{
				Url:   "https://some.url",
				Token: monad.Nil[string](),
			}),
		})

		assert.Nil(t, err)
		assert.Equal(t, string(app.ID()), id)
		assert.HasLength(t, 1, dispatcher.Signals())
		configured := assert.Is[domain.AppVersionControlConfigured](t, dispatcher.Signals()[0])
		assert.Equal(t, app.ID(), configured.ID)
		assert.Equal(t, url, configured.Config.Url())
		assert.False(t, configured.Config.Token().HasValue())
	})

	t.Run("should update the vcs token", func(t *testing.T) {
		user := authfixture.User()
		target := fixture.Target(fixture.WithTargetCreatedBy(user.ID()))
		app := fixture.App(
			fixture.WithAppCreatedBy(user.ID()),
			fixture.WithEnvironmentConfig(
				domain.NewEnvironmentConfig(target.ID()),
				domain.NewEnvironmentConfig(target.ID()),
			),
		)
		url := must.Panic(domain.UrlFrom("https://some.url"))
		vcs := domain.NewVersionControl(url)
		vcs.Authenticated("a token")
		assert.Nil(t, app.UseVersionControl(vcs))
		handler, ctx, dispatcher := arrange(t,
			fixture.WithUsers(&user),
			fixture.WithTargets(&target),
			fixture.WithApps(&app),
		)

		id, err := handler(ctx, update_app.Command{
			ID: string(app.ID()),
			VersionControl: monad.PatchValue(update_app.VersionControl{
				Url:   "https://some.url",
				Token: monad.PatchValue("new token"),
			}),
		})

		assert.Nil(t, err)
		assert.Equal(t, string(app.ID()), id)
		assert.HasLength(t, 1, dispatcher.Signals())
		configured := assert.Is[domain.AppVersionControlConfigured](t, dispatcher.Signals()[0])
		assert.Equal(t, app.ID(), configured.ID)
		assert.Equal(t, url, configured.Config.Url())
		assert.Equal(t, "new token", configured.Config.Token().Get(""))
	})
}
