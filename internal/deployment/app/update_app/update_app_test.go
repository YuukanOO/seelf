package update_app_test

import (
	"context"
	"testing"

	auth "github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/internal/deployment/app/update_app"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/deployment/infra/memory"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/must"
	"github.com/YuukanOO/seelf/pkg/testutil"
	"github.com/YuukanOO/seelf/pkg/validate"
)

func Test_UpdateApp(t *testing.T) {
	production := domain.NewEnvironmentConfig("1")
	production.HasEnvironmentVariables(domain.ServicesEnv{"app": {"DEBUG": "false"}})
	staging := domain.NewEnvironmentConfig("1")
	staging.HasEnvironmentVariables(domain.ServicesEnv{"app": {"DEBUG": "false"}})
	ctx := auth.WithUserID(context.Background(), "some-uid")

	sut := func(existingApps ...*domain.App) bus.RequestHandler[string, update_app.Command] {
		store := memory.NewAppsStore(existingApps...)
		return update_app.Handler(store, store)
	}

	t.Run("should require a valid application id", func(t *testing.T) {
		uc := sut()
		id, err := uc(ctx, update_app.Command{})

		testutil.ErrorIs(t, apperr.ErrNotFound, err)
		testutil.Equals(t, "", id)
	})

	t.Run("should update nothing if no fields are provided", func(t *testing.T) {
		a := must.Panic(domain.NewApp("my-app",
			domain.NewEnvironmentConfigRequirement(domain.NewEnvironmentConfig("1"), true, true),
			domain.NewEnvironmentConfigRequirement(domain.NewEnvironmentConfig("1"), true, true), "some-uid"))
		uc := sut(&a)

		id, err := uc(ctx, update_app.Command{
			ID: string(a.ID()),
		})

		testutil.IsNil(t, err)
		testutil.Equals(t, string(a.ID()), id)
		testutil.HasNEvents(t, &a, 1)
	})

	t.Run("should validate new target naming availability", func(t *testing.T) {
		a1 := must.Panic(domain.NewApp("my-app",
			domain.NewEnvironmentConfigRequirement(domain.NewEnvironmentConfig("1"), true, true),
			domain.NewEnvironmentConfigRequirement(domain.NewEnvironmentConfig("2"), true, true), "some-uid"))
		a2 := must.Panic(domain.NewApp("my-app",
			domain.NewEnvironmentConfigRequirement(domain.NewEnvironmentConfig("3"), true, true),
			domain.NewEnvironmentConfigRequirement(domain.NewEnvironmentConfig("4"), true, true), "some-uid"))
		uc := sut(&a1, &a2)

		_, err := uc(ctx, update_app.Command{
			ID: string(a2.ID()),
			Production: monad.Value(update_app.EnvironmentConfig{
				Target: "1",
			}),
			Staging: monad.Value(update_app.EnvironmentConfig{
				Target: "2",
			}),
		})

		testutil.ErrorIs(t, validate.ErrValidationFailed, err)
		validationErr, ok := apperr.As[validate.FieldErrors](err)
		testutil.IsTrue(t, ok)
		testutil.ErrorIs(t, domain.ErrAppNameAlreadyTaken, validationErr["production.target"])
		testutil.ErrorIs(t, domain.ErrAppNameAlreadyTaken, validationErr["staging.target"])
	})

	t.Run("should remove an application env variables", func(t *testing.T) {
		a := must.Panic(domain.NewApp("an-app",
			domain.NewEnvironmentConfigRequirement(production, true, true),
			domain.NewEnvironmentConfigRequirement(staging, true, true),
			"uid",
		))

		uc := sut(&a)

		id, err := uc(ctx, update_app.Command{
			ID: string(a.ID()),
			Production: monad.Value(update_app.EnvironmentConfig{
				Target: "new-production-target",
			}),
			Staging: monad.Value(update_app.EnvironmentConfig{
				Target: "new-staging-target",
			}),
		})

		testutil.IsNil(t, err)
		testutil.Equals(t, string(a.ID()), id)
		testutil.HasNEvents(t, &a, 3)

		evt := testutil.EventIs[domain.AppEnvChanged](t, &a, 1)

		testutil.Equals(t, domain.Production, evt.Environment)
		testutil.Equals(t, "new-production-target", evt.Config.Target())
		testutil.IsFalse(t, evt.Config.Vars().HasValue())

		evt = testutil.EventIs[domain.AppEnvChanged](t, &a, 2)

		testutil.Equals(t, domain.Staging, evt.Environment)
		testutil.Equals(t, "new-staging-target", evt.Config.Target())
		testutil.IsFalse(t, evt.Config.Vars().HasValue())
	})

	t.Run("should update an application env variables", func(t *testing.T) {
		a := must.Panic(domain.NewApp("an-app",
			domain.NewEnvironmentConfigRequirement(production, true, true),
			domain.NewEnvironmentConfigRequirement(staging, true, true),
			"uid",
		))

		uc := sut(&a)

		id, err := uc(ctx, update_app.Command{
			ID: string(a.ID()),
			Production: monad.Value(update_app.EnvironmentConfig{
				Target: "new-production-target",
				Vars: monad.Value(map[string]map[string]string{
					"app": {"OTHER": "value"},
				}),
			}),
			Staging: monad.Value(update_app.EnvironmentConfig{
				Target: "new-staging-target",
				Vars: monad.Value(map[string]map[string]string{
					"app": {"SOMETHING": "else"},
				}),
			}),
		})

		testutil.IsNil(t, err)
		testutil.Equals(t, string(a.ID()), id)
		testutil.HasNEvents(t, &a, 3)

		evt := testutil.EventIs[domain.AppEnvChanged](t, &a, 1)

		testutil.Equals(t, domain.Production, evt.Environment)
		testutil.Equals(t, "new-production-target", evt.Config.Target())
		testutil.DeepEquals(t, domain.ServicesEnv{
			"app": {"OTHER": "value"},
		}, evt.Config.Vars().MustGet())

		evt = testutil.EventIs[domain.AppEnvChanged](t, &a, 2)

		testutil.Equals(t, domain.Staging, evt.Environment)
		testutil.Equals(t, "new-staging-target", evt.Config.Target())
		testutil.DeepEquals(t, domain.ServicesEnv{
			"app": {"SOMETHING": "else"},
		}, evt.Config.Vars().MustGet())
	})

	t.Run("should require valid vcs inputs", func(t *testing.T) {
		uc := sut()
		id, err := uc(ctx, update_app.Command{
			ID: "an-app",
			VersionControl: monad.PatchValue(update_app.VersionControl{
				Url: "invalid-url",
			}),
		})

		testutil.ErrorIs(t, validate.ErrValidationFailed, err)
		testutil.Equals(t, "", id)
	})

	t.Run("should fail if trying to update an app being deleted", func(t *testing.T) {
		a := must.Panic(domain.NewApp("an-app",
			domain.NewEnvironmentConfigRequirement(production, true, true),
			domain.NewEnvironmentConfigRequirement(staging, true, true),
			"uid",
		))
		a.RequestCleanup("uid")

		uc := sut(&a)

		_, err := uc(ctx, update_app.Command{
			ID: string(a.ID()),
			VersionControl: monad.PatchValue(update_app.VersionControl{
				Url: "https://some.url",
			}),
		})

		testutil.ErrorIs(t, domain.ErrAppCleanupRequested, err)
	})

	t.Run("should fail if trying to add a vcs config without an url defined", func(t *testing.T) {
		a := must.Panic(domain.NewApp("an-app",
			domain.NewEnvironmentConfigRequirement(production, true, true),
			domain.NewEnvironmentConfigRequirement(staging, true, true),
			"uid",
		))

		uc := sut(&a)

		id, err := uc(ctx, update_app.Command{
			ID:             string(a.ID()),
			VersionControl: monad.PatchValue(update_app.VersionControl{}),
		})

		testutil.ErrorIs(t, validate.ErrValidationFailed, err)
		testutil.Equals(t, "", id)
	})

	t.Run("should remove the vcs config if nil given", func(t *testing.T) {
		a := must.Panic(domain.NewApp("an-app",
			domain.NewEnvironmentConfigRequirement(production, true, true),
			domain.NewEnvironmentConfigRequirement(staging, true, true),
			"uid",
		))
		url := must.Panic(domain.UrlFrom("https://some.url"))
		a.UseVersionControl(domain.NewVersionControl(url))

		uc := sut(&a)

		id, err := uc(ctx, update_app.Command{
			ID:             string(a.ID()),
			VersionControl: monad.Nil[update_app.VersionControl](),
		})

		testutil.IsNil(t, err)
		testutil.Equals(t, string(a.ID()), id)
		testutil.HasNEvents(t, &a, 3)
		testutil.EventIs[domain.AppVersionControlRemoved](t, &a, 2)
	})

	t.Run("should update the vcs url", func(t *testing.T) {
		a := must.Panic(domain.NewApp("an-app",
			domain.NewEnvironmentConfigRequirement(production, true, true),
			domain.NewEnvironmentConfigRequirement(staging, true, true),
			"uid",
		))
		url := must.Panic(domain.UrlFrom("https://some.url"))
		vcs := domain.NewVersionControl(url)
		vcs.Authenticated("a token")
		a.UseVersionControl(vcs)

		uc := sut(&a)

		id, err := uc(ctx, update_app.Command{
			ID: string(a.ID()),
			VersionControl: monad.PatchValue(update_app.VersionControl{
				Url: "https://some.other.url",
			}),
		})

		testutil.IsNil(t, err)
		testutil.Equals(t, string(a.ID()), id)
		testutil.HasNEvents(t, &a, 3)
		evt := testutil.EventIs[domain.AppVersionControlConfigured](t, &a, 2)
		testutil.Equals(t, "https://some.other.url", evt.Config.Url().String())
		testutil.Equals(t, "a token", evt.Config.Token().MustGet())
	})

	t.Run("should remove the vcs token", func(t *testing.T) {
		a := must.Panic(domain.NewApp("an-app",
			domain.NewEnvironmentConfigRequirement(production, true, true),
			domain.NewEnvironmentConfigRequirement(staging, true, true),
			"uid",
		))
		url := must.Panic(domain.UrlFrom("https://some.url"))
		vcs := domain.NewVersionControl(url)
		vcs.Authenticated("a token")
		a.UseVersionControl(vcs)

		uc := sut(&a)

		id, err := uc(ctx, update_app.Command{
			ID: string(a.ID()),
			VersionControl: monad.PatchValue(update_app.VersionControl{
				Url:   "https://some.url",
				Token: monad.Nil[string](),
			}),
		})

		testutil.IsNil(t, err)
		testutil.Equals(t, string(a.ID()), id)
		testutil.HasNEvents(t, &a, 3)
		evt := testutil.EventIs[domain.AppVersionControlConfigured](t, &a, 2)
		testutil.Equals(t, "https://some.url", evt.Config.Url().String())
		testutil.IsFalse(t, evt.Config.Token().HasValue())
	})

	t.Run("should update the vcs token", func(t *testing.T) {
		a := must.Panic(domain.NewApp("an-app",
			domain.NewEnvironmentConfigRequirement(production, true, true),
			domain.NewEnvironmentConfigRequirement(staging, true, true),
			"uid",
		))
		url := must.Panic(domain.UrlFrom("https://some.url"))
		vcs := domain.NewVersionControl(url)
		vcs.Authenticated("a token")
		a.UseVersionControl(vcs)

		uc := sut(&a)

		id, err := uc(ctx, update_app.Command{
			ID: string(a.ID()),
			VersionControl: monad.PatchValue(update_app.VersionControl{
				Url:   "https://some.url",
				Token: monad.PatchValue("new token"),
			}),
		})

		testutil.IsNil(t, err)
		testutil.Equals(t, string(a.ID()), id)
		testutil.HasNEvents(t, &a, 3)
		evt := testutil.EventIs[domain.AppVersionControlConfigured](t, &a, 2)
		testutil.Equals(t, "https://some.url", evt.Config.Url().String())
		testutil.Equals(t, "new token", evt.Config.Token().Get(""))
	})
}
