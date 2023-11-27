package command

import (
	"context"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/validation"
	"github.com/YuukanOO/seelf/pkg/validation/strings"
)

type (
	UpdateAppCommand struct {
		ID  string                                               `json:"-"`
		VCS monad.Patch[VCSConfigUpdate]                         `json:"vcs"`
		Env monad.Patch[map[string]map[string]map[string]string] `json:"env"`
	}

	VCSConfigUpdate struct {
		Url   monad.Maybe[string] `json:"url"`
		Token monad.Patch[string] `json:"token"`
	}
)

func UpdateApp(
	reader domain.AppsReader,
	writer domain.AppsWriter,
) func(context.Context, UpdateAppCommand) error {
	return func(ctx context.Context, cmd UpdateAppCommand) error {
		var (
			envs domain.EnvironmentsEnv
			url  domain.Url
		)

		if err := validation.Check(validation.Of{
			"vcs": validation.Patch(cmd.VCS, func(config VCSConfigUpdate) error {
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
			return err
		}

		app, err := reader.GetByID(ctx, domain.AppID(cmd.ID))

		if err != nil {
			return err
		}

		if vcsPatch, isSet := cmd.VCS.TryGet(); isSet {
			if vcsUpdate, hasValue := vcsPatch.TryGet(); hasValue {
				// No VCS configured on the app and no url given
				if !app.VCS().HasValue() && !vcsUpdate.Url.HasValue() {
					return domain.ErrVCSNotConfigured
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

		return writer.Write(ctx, &app)
	}
}
