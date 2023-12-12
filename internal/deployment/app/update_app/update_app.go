package update_app

import (
	"context"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/validation"
	"github.com/YuukanOO/seelf/pkg/validation/strings"
)

type (
	// Update an existing application.
	Command struct {
		bus.Command[string]

		ID  string                                               `json:"-"`
		VCS monad.Patch[VCSConfig]                               `json:"vcs"`
		Env monad.Patch[map[string]map[string]map[string]string] `json:"env"`
	}

	VCSConfig struct {
		Url   monad.Maybe[string] `json:"url"`
		Token monad.Patch[string] `json:"token"`
	}
)

func (Command) Name_() string { return "deployment.command.update_app" }

func Handler(
	reader domain.AppsReader,
	writer domain.AppsWriter,
) bus.RequestHandler[string, Command] {
	return func(ctx context.Context, cmd Command) (string, error) {
		var (
			envs domain.EnvironmentsEnv
			url  domain.Url
		)

		if err := validation.Check(validation.Of{
			"vcs": validation.Patch(cmd.VCS, func(config VCSConfig) error {
				return validation.Check(validation.Of{
					"url": validation.Maybe(config.Url, func(urlValue string) error {
						return validation.Value(urlValue, &url, domain.UrlFrom)
					}),
					"token": validation.Patch(config.Token, strings.Required),
				})
			}),
			"env": validation.Patch(cmd.Env, func(envmap map[string]map[string]map[string]string) error {
				return validation.Value(envmap, &envs, domain.EnvironmentsEnvFrom)
			}),
		}); err != nil {
			return "", err
		}

		app, err := reader.GetByID(ctx, domain.AppID(cmd.ID))

		if err != nil {
			return "", err
		}

		if vcsPatch, isSet := cmd.VCS.TryGet(); isSet {
			if vcsUpdate, hasValue := vcsPatch.TryGet(); hasValue {
				// No VCS configured on the app and no url given
				if !app.VCS().HasValue() && !vcsUpdate.Url.HasValue() {
					return "", domain.ErrVCSNotConfigured
				}

				vcs := app.VCS().Get(domain.NewVCSConfig(url))

				if vcsUpdate.Url.HasValue() {
					vcs = vcs.WithUrl(url)
				}

				if tokenPatch, isSet := vcsUpdate.Token.TryGet(); isSet {
					if token, hasValue := tokenPatch.TryGet(); hasValue {
						vcs = vcs.Authenticated(token)
					} else {
						vcs = vcs.Public()
					}
				}

				app.UseVersionControl(vcs)
			} else {
				app.RemoveVersionControl()
			}
		}

		if cmd.Env.IsSet() {
			if cmd.Env.IsNil() {
				app.RemoveEnvironmentVariables()
			} else {
				app.HasEnvironmentVariables(envs)
			}
		}

		if err = writer.Write(ctx, &app); err != nil {
			return "", err
		}

		return cmd.ID, nil
	}
}
