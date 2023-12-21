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
	"github.com/YuukanOO/seelf/pkg/testutil"
	"github.com/YuukanOO/seelf/pkg/validation"
)

func Test_UpdateApp(t *testing.T) {
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
		a := domain.NewApp("an-app", "uid")
		uc := sut(&a)

		id, err := uc(ctx, update_app.Command{
			ID: string(a.ID()),
		})

		testutil.IsNil(t, err)
		testutil.Equals(t, string(a.ID()), id)
		testutil.HasNEvents(t, &a, 1)
	})

	t.Run("should require valid application env variables", func(t *testing.T) {
		uc := sut()
		id, err := uc(ctx, update_app.Command{
			ID: "an-app",
			Env: monad.PatchValue(map[string]map[string]map[string]string{
				"invalidenv": {},
			}),
		})

		testutil.ErrorIs(t, validation.ErrValidationFailed, err)
		testutil.Equals(t, "", id)
	})

	t.Run("should remove an application env variables", func(t *testing.T) {
		a := domain.NewApp("an-app", "uid")
		a.HasEnvironmentVariables(domain.EnvironmentsEnv{"production": {"app": {"DEBUG": "false"}}})
		uc := sut(&a)

		id, err := uc(ctx, update_app.Command{
			ID:  string(a.ID()),
			Env: monad.Nil[map[string]map[string]map[string]string](),
		})

		testutil.IsNil(t, err)
		testutil.Equals(t, string(a.ID()), id)
		testutil.HasNEvents(t, &a, 3)
		testutil.EventIs[domain.AppEnvRemoved](t, &a, 2)
	})

	t.Run("should update an application env variables", func(t *testing.T) {
		a := domain.NewApp("an-app", "uid")
		a.HasEnvironmentVariables(domain.EnvironmentsEnv{"production": {"app": {"DEBUG": "false"}}})
		uc := sut(&a)

		id, err := uc(ctx, update_app.Command{
			ID: string(a.ID()),
			Env: monad.PatchValue(map[string]map[string]map[string]string{
				"production": {"app": {"NEW": "value"}},
			}),
		})

		testutil.IsNil(t, err)
		testutil.Equals(t, string(a.ID()), id)
		testutil.HasNEvents(t, &a, 3)
		testutil.EventIs[domain.AppEnvChanged](t, &a, 2)
	})

	t.Run("should require valid vcs inputs", func(t *testing.T) {
		uc := sut()
		id, err := uc(ctx, update_app.Command{
			ID: "an-app",
			VCS: monad.PatchValue(update_app.VCSConfig{
				Url: monad.Value("invalid-url"),
			}),
		})

		testutil.ErrorIs(t, validation.ErrValidationFailed, err)
		testutil.Equals(t, "", id)
	})

	t.Run("should fail if trying to add a vcs config without an url defined", func(t *testing.T) {
		a := domain.NewApp("an-app", "uid")
		uc := sut(&a)

		id, err := uc(ctx, update_app.Command{
			ID:  string(a.ID()),
			VCS: monad.PatchValue(update_app.VCSConfig{}),
		})

		testutil.ErrorIs(t, domain.ErrVCSNotConfigured, err)
		testutil.Equals(t, "", id)
	})

	t.Run("should remove the vcs config", func(t *testing.T) {
		a := domain.NewApp("an-app", "uid")
		url, _ := domain.UrlFrom("https://some.url")
		a.UseVersionControl(domain.NewVCSConfig(url))
		uc := sut(&a)

		id, err := uc(ctx, update_app.Command{
			ID:  string(a.ID()),
			VCS: monad.Nil[update_app.VCSConfig](),
		})

		testutil.IsNil(t, err)
		testutil.Equals(t, string(a.ID()), id)
		testutil.HasNEvents(t, &a, 3)
		testutil.EventIs[domain.AppVCSRemoved](t, &a, 2)
	})

	t.Run("should keep the vcs url", func(t *testing.T) {
		a := domain.NewApp("an-app", "uid")
		url, _ := domain.UrlFrom("https://some.url")
		a.UseVersionControl(domain.NewVCSConfig(url))
		uc := sut(&a)

		id, err := uc(ctx, update_app.Command{
			ID: string(a.ID()),
			VCS: monad.PatchValue(update_app.VCSConfig{
				Url: monad.None[string](),
			}),
		})

		testutil.IsNil(t, err)
		testutil.Equals(t, string(a.ID()), id)
		testutil.HasNEvents(t, &a, 2)
	})

	t.Run("should update the vcs url", func(t *testing.T) {
		a := domain.NewApp("an-app", "uid")
		url, _ := domain.UrlFrom("https://some.url")
		a.UseVersionControl(domain.NewVCSConfig(url).Authenticated("a token"))
		uc := sut(&a)

		id, err := uc(ctx, update_app.Command{
			ID: string(a.ID()),
			VCS: monad.PatchValue(update_app.VCSConfig{
				Url: monad.Value("https://some.other.url"),
			}),
		})

		testutil.IsNil(t, err)
		testutil.Equals(t, string(a.ID()), id)
		testutil.HasNEvents(t, &a, 3)
		evt := testutil.EventIs[domain.AppVCSConfigured](t, &a, 2)
		testutil.Equals(t, "https://some.other.url", evt.Config.Url().String())
		testutil.Equals(t, "a token", evt.Config.Token().Get(""))
	})

	t.Run("should remove the vcs token", func(t *testing.T) {
		a := domain.NewApp("an-app", "uid")
		url, _ := domain.UrlFrom("https://some.url")
		a.UseVersionControl(domain.NewVCSConfig(url).Authenticated("a token"))
		uc := sut(&a)

		id, err := uc(ctx, update_app.Command{
			ID: string(a.ID()),
			VCS: monad.PatchValue(update_app.VCSConfig{
				Token: monad.Nil[string](),
			}),
		})

		testutil.IsNil(t, err)
		testutil.Equals(t, string(a.ID()), id)
		testutil.HasNEvents(t, &a, 3)
		evt := testutil.EventIs[domain.AppVCSConfigured](t, &a, 2)
		testutil.Equals(t, "https://some.url", evt.Config.Url().String())
		testutil.IsFalse(t, evt.Config.Token().HasValue())
	})

	t.Run("should update the vcs token", func(t *testing.T) {
		a := domain.NewApp("an-app", "uid")
		url, _ := domain.UrlFrom("https://some.url")
		a.UseVersionControl(domain.NewVCSConfig(url).Authenticated("a token"))
		uc := sut(&a)

		id, err := uc(ctx, update_app.Command{
			ID: string(a.ID()),
			VCS: monad.PatchValue(update_app.VCSConfig{
				Token: monad.PatchValue("new token"),
			}),
		})

		testutil.IsNil(t, err)
		testutil.Equals(t, string(a.ID()), id)
		testutil.HasNEvents(t, &a, 3)
		evt := testutil.EventIs[domain.AppVCSConfigured](t, &a, 2)
		testutil.Equals(t, "https://some.url", evt.Config.Url().String())
		testutil.Equals(t, "new token", evt.Config.Token().Get(""))
	})
}
