package command_test

import (
	"context"
	"testing"

	auth "github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/internal/deployment/app/command"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/deployment/infra/memory"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/testutil"
	"github.com/YuukanOO/seelf/pkg/validation"
)

func Test_UpdateApp(t *testing.T) {
	ctx := auth.WithUserID(context.Background(), "some-uid")
	update := func(existingApps ...*domain.App) func(context.Context, command.UpdateAppCommand) error {
		store := memory.NewAppsStore(existingApps...)
		return command.UpdateApp(store, store)
	}

	t.Run("should require a valid application id", func(t *testing.T) {
		uc := update()
		err := uc(ctx, command.UpdateAppCommand{})

		testutil.ErrorIs(t, apperr.ErrNotFound, err)
	})

	t.Run("should update nothing if no fields are provided", func(t *testing.T) {
		app := domain.NewApp("an-app", "uid")
		uc := update(&app)

		err := uc(ctx, command.UpdateAppCommand{
			ID: string(app.ID()),
		})

		testutil.IsNil(t, err)
		testutil.HasNEvents(t, &app, 1)
	})

	t.Run("should require valid application env variables", func(t *testing.T) {
		uc := update()
		err := uc(ctx, command.UpdateAppCommand{
			ID: "an-app",
			Env: monad.PatchValue(map[string]map[string]map[string]string{
				"invalidenv": {},
			}),
		})

		testutil.ErrorIs(t, validation.ErrValidationFailed, err)
	})

	t.Run("should remove an application env variables", func(t *testing.T) {
		app := domain.NewApp("an-app", "uid")
		app.HasEnvironmentVariables(domain.EnvironmentsEnv{"production": {"app": {"DEBUG": "false"}}})
		uc := update(&app)

		err := uc(ctx, command.UpdateAppCommand{
			ID:  string(app.ID()),
			Env: monad.Nil[map[string]map[string]map[string]string](),
		})

		testutil.IsNil(t, err)
		testutil.HasNEvents(t, &app, 3)
		testutil.EventIs[domain.AppEnvRemoved](t, &app, 2)
	})

	t.Run("should update an application env variables", func(t *testing.T) {
		app := domain.NewApp("an-app", "uid")
		app.HasEnvironmentVariables(domain.EnvironmentsEnv{"production": {"app": {"DEBUG": "false"}}})
		uc := update(&app)

		err := uc(ctx, command.UpdateAppCommand{
			ID: string(app.ID()),
			Env: monad.PatchValue(map[string]map[string]map[string]string{
				"production": {"app": {"NEW": "value"}},
			}),
		})

		testutil.IsNil(t, err)
		testutil.HasNEvents(t, &app, 3)
		testutil.EventIs[domain.AppEnvChanged](t, &app, 2)
	})

	t.Run("should require valid vcs inputs", func(t *testing.T) {
		uc := update()
		err := uc(ctx, command.UpdateAppCommand{
			ID: "an-app",
			VCS: monad.PatchValue(command.VCSConfigUpdate{
				Url: monad.Value("invalid-url"),
			}),
		})

		testutil.ErrorIs(t, validation.ErrValidationFailed, err)
	})

	t.Run("should fail if trying to add a vcs config without an url defined", func(t *testing.T) {
		app := domain.NewApp("an-app", "uid")
		uc := update(&app)

		err := uc(ctx, command.UpdateAppCommand{
			ID:  string(app.ID()),
			VCS: monad.PatchValue(command.VCSConfigUpdate{}),
		})

		testutil.ErrorIs(t, domain.ErrVCSNotConfigured, err)
	})

	t.Run("should remove the vcs config", func(t *testing.T) {
		app := domain.NewApp("an-app", "uid")
		url, _ := domain.UrlFrom("https://some.url")
		app.UseVersionControl(domain.NewVCSConfig(url))
		uc := update(&app)

		err := uc(ctx, command.UpdateAppCommand{
			ID:  string(app.ID()),
			VCS: monad.Nil[command.VCSConfigUpdate](),
		})

		testutil.IsNil(t, err)
		testutil.HasNEvents(t, &app, 3)
		testutil.EventIs[domain.AppVCSRemoved](t, &app, 2)
	})

	t.Run("should keep the vcs url", func(t *testing.T) {
		app := domain.NewApp("an-app", "uid")
		url, _ := domain.UrlFrom("https://some.url")
		app.UseVersionControl(domain.NewVCSConfig(url))
		uc := update(&app)

		err := uc(ctx, command.UpdateAppCommand{
			ID: string(app.ID()),
			VCS: monad.PatchValue(command.VCSConfigUpdate{
				Url: monad.None[string](),
			}),
		})

		testutil.IsNil(t, err)
		testutil.HasNEvents(t, &app, 2)
	})

	t.Run("should update the vcs url", func(t *testing.T) {
		app := domain.NewApp("an-app", "uid")
		url, _ := domain.UrlFrom("https://some.url")
		app.UseVersionControl(domain.NewVCSConfig(url).Authenticated("a token"))
		uc := update(&app)

		err := uc(ctx, command.UpdateAppCommand{
			ID: string(app.ID()),
			VCS: monad.PatchValue(command.VCSConfigUpdate{
				Url: monad.Value("https://some.other.url"),
			}),
		})

		testutil.IsNil(t, err)
		testutil.HasNEvents(t, &app, 3)
		evt := testutil.EventIs[domain.AppVCSConfigured](t, &app, 2)
		testutil.Equals(t, "https://some.other.url", evt.Config.Url().String())
		testutil.Equals(t, "a token", evt.Config.Token().Get(""))
	})

	t.Run("should remove the vcs token", func(t *testing.T) {
		app := domain.NewApp("an-app", "uid")
		url, _ := domain.UrlFrom("https://some.url")
		app.UseVersionControl(domain.NewVCSConfig(url).Authenticated("a token"))
		uc := update(&app)

		err := uc(ctx, command.UpdateAppCommand{
			ID: string(app.ID()),
			VCS: monad.PatchValue(command.VCSConfigUpdate{
				Token: monad.Nil[string](),
			}),
		})

		testutil.IsNil(t, err)
		testutil.HasNEvents(t, &app, 3)
		evt := testutil.EventIs[domain.AppVCSConfigured](t, &app, 2)
		testutil.Equals(t, "https://some.url", evt.Config.Url().String())
		testutil.IsFalse(t, evt.Config.Token().HasValue())
	})

	t.Run("should update the vcs token", func(t *testing.T) {
		app := domain.NewApp("an-app", "uid")
		url, _ := domain.UrlFrom("https://some.url")
		app.UseVersionControl(domain.NewVCSConfig(url).Authenticated("a token"))
		uc := update(&app)

		err := uc(ctx, command.UpdateAppCommand{
			ID: string(app.ID()),
			VCS: monad.PatchValue(command.VCSConfigUpdate{
				Token: monad.PatchValue("new token"),
			}),
		})

		testutil.IsNil(t, err)
		testutil.HasNEvents(t, &app, 3)
		evt := testutil.EventIs[domain.AppVCSConfigured](t, &app, 2)
		testutil.Equals(t, "https://some.url", evt.Config.Url().String())
		testutil.Equals(t, "new token", evt.Config.Token().Get(""))
	})
}
