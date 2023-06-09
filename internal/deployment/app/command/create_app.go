package command

import (
	"context"

	auth "github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/validation"
	"github.com/YuukanOO/seelf/pkg/validation/strings"
)

type (
	CreateAppCommand struct {
		Name string                                               `json:"name"`
		VCS  monad.Maybe[VCSConfig]                               `json:"vcs"`
		Env  monad.Maybe[map[string]map[string]map[string]string] `json:"env"` // This is not so sweet but hey!
	}

	VCSConfig struct {
		Url   string              `json:"url"`
		Token monad.Maybe[string] `json:"token"`
	}
)

// Create a new application.
func CreateApp(
	reader domain.AppsReader,
	writer domain.AppsWriter,
) func(context.Context, CreateAppCommand) (string, error) {
	return func(ctx context.Context, cmd CreateAppCommand) (string, error) {
		var (
			appname domain.AppName
			envs    domain.EnvironmentsEnv
			url     domain.Url
		)

		if err := validation.Check(validation.Of{
			"name": validation.Value(cmd.Name, &appname, domain.AppNameFrom),
			"vcs": validation.Maybe(cmd.VCS, func(config VCSConfig) error {
				return validation.Check(validation.Of{
					"url": validation.Value(config.Url, &url, domain.UrlFrom),
					"token": validation.Maybe(config.Token, func(tokenValue string) error {
						return validation.Is(tokenValue, strings.Required)
					}),
				})
			}),
			"env": validation.Maybe(cmd.Env, func(envmap map[string]map[string]map[string]string) error {
				return validation.Value(envmap, &envs, domain.EnvironmentsEnvFrom)
			}),
		}); err != nil {
			return "", err
		}

		uniqueName, err := reader.IsNameUnique(ctx, appname)

		if err != nil {
			return "", validation.WrapIfAppErr(err, "name")
		}

		app := domain.NewApp(uniqueName, auth.CurrentUser(ctx).MustGet())

		if cmd.VCS.HasValue() {
			vcs := domain.NewVCSConfig(url)
			cmdVCS := cmd.VCS.MustGet()

			if cmdVCS.Token.HasValue() {
				vcs = vcs.Authenticated(cmdVCS.Token.MustGet())
			}

			app.UseVersionControl(vcs)
		}

		if cmd.Env.HasValue() {
			app.HasEnvironmentVariables(envs)
		}

		if err := writer.Write(ctx, &app); err != nil {
			return "", err
		}

		return string(app.ID()), nil
	}
}
