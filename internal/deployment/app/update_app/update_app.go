package update_app

import (
	"context"

	"github.com/YuukanOO/seelf/internal/deployment/app/create_app"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/validate"
	"github.com/YuukanOO/seelf/pkg/validate/strings"
)

type (
	// Update an existing application.
	Command struct {
		bus.Command[string]

		ID             string                         `json:"-"`
		VersionControl monad.Patch[VersionControl]    `json:"version_control"`
		Production     monad.Maybe[EnvironmentConfig] `json:"production"`
		Staging        monad.Maybe[EnvironmentConfig] `json:"staging"`
	}

	EnvironmentConfig create_app.EnvironmentConfig

	VersionControl struct {
		Url   string              `json:"url"`
		Token monad.Patch[string] `json:"token"`
	}
)

func (Command) Name_() string { return "deployment.command.update_app" }

func Handler(
	reader domain.AppsReader,
	writer domain.AppsWriter,
) bus.RequestHandler[string, Command] {
	return func(ctx context.Context, cmd Command) (string, error) {
		var url domain.Url

		if err := validate.Struct(validate.Of{
			"version_control": validate.Patch(cmd.VersionControl, func(config VersionControl) error {
				return validate.Struct(validate.Of{
					"url":   validate.Value(config.Url, &url, domain.UrlFrom),
					"token": validate.Patch(config.Token, strings.Required),
				})
			}),
			"production": validate.Maybe(cmd.Production, func(conf EnvironmentConfig) error {
				return validate.Struct(validate.Of{
					"target": validate.Field(conf.Target, strings.Required),
				})
			}),
			"staging": validate.Maybe(cmd.Staging, func(conf EnvironmentConfig) error {
				return validate.Struct(validate.Of{
					"target": validate.Field(conf.Target, strings.Required),
				})
			}),
		}); err != nil {
			return "", err
		}

		app, err := reader.GetByID(ctx, domain.AppID(cmd.ID))

		if err != nil {
			return "", err
		}

		// Determine the availability of updated targets
		var productionConfig, stagingConfig monad.Maybe[domain.EnvironmentConfig]

		if conf, isUpdated := cmd.Production.TryGet(); isUpdated {
			productionConfig.Set(create_app.BuildEnvironmentConfig(domain.TargetID(conf.Target), conf.Vars))
		}

		if conf, isUpdated := cmd.Staging.TryGet(); isUpdated {
			stagingConfig.Set(create_app.BuildEnvironmentConfig(domain.TargetID(conf.Target), conf.Vars))
		}

		productionRequirement, stagingRequirement, err := reader.CheckAppNamingAvailabilityByID(ctx, app.ID(), productionConfig, stagingConfig)

		if err != nil {
			return "", err
		}

		if err = validate.Struct(validate.Of{
			"production.target": validate.If(productionConfig.HasValue(), productionRequirement.Error),
			"staging.target":    validate.If(stagingConfig.HasValue(), stagingRequirement.Error),
		}); err != nil {
			return "", err
		}

		if vcsPatch, isSet := cmd.VersionControl.TryGet(); isSet {
			if vcsUpdate, hasValue := vcsPatch.TryGet(); hasValue {
				// Take the existing vcs as a reference so the token is not modified if not provided at all
				vcs := app.VersionControl().Get(domain.NewVersionControl(url))
				vcs.HasUrl(url)

				if tokenPatch, isSet := vcsUpdate.Token.TryGet(); isSet {
					if token, hasValue := tokenPatch.TryGet(); hasValue {
						vcs.Authenticated(token)
					} else {
						vcs.Public()
					}
				}

				err = app.UseVersionControl(vcs)
			} else {
				err = app.RemoveVersionControl()
			}

			if err != nil {
				return "", err
			}
		}

		if productionConfig.HasValue() {
			if err = app.HasProductionConfig(productionRequirement); err != nil {
				return "", err
			}
		}

		if stagingConfig.HasValue() {
			if err = app.HasStagingConfig(stagingRequirement); err != nil {
				return "", err
			}
		}

		if err = writer.Write(ctx, &app); err != nil {
			return "", err
		}

		return cmd.ID, nil
	}
}
