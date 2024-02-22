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

		ID         string                         `json:"-"`
		VCS        monad.Patch[VCSConfig]         `json:"vcs"`
		Production monad.Maybe[EnvironmentConfig] `json:"production"`
		Staging    monad.Maybe[EnvironmentConfig] `json:"staging"`
	}

	EnvironmentConfig create_app.EnvironmentConfig

	VCSConfig struct {
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
			"vcs": validate.Patch(cmd.VCS, func(config VCSConfig) error {
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

		// Check availability of new targets if set to report errors early
		var (
			hasProductionConfigUpdate    bool
			productionConfigUpdate       EnvironmentConfig
			productionTarget             domain.TargetID
			productionTargetAvailability domain.TargetAppNamingAvailability
		)

		if productionConfigUpdate, hasProductionConfigUpdate = cmd.Production.TryGet(); hasProductionConfigUpdate {
			productionTarget = domain.TargetID(productionConfigUpdate.Target)
			productionTargetAvailability, err = reader.GetTargetAppNamingAvailability(ctx, app.ID(), domain.Production, productionTarget)

			if err != nil {
				return "", err
			}
		}

		var (
			stagingTarget             domain.TargetID
			hasStagingConfigUpdate    bool
			stagingConfigUpdate       EnvironmentConfig
			stagingTargetAvailability domain.TargetAppNamingAvailability
		)

		if stagingConfigUpdate, hasStagingConfigUpdate = cmd.Staging.TryGet(); hasStagingConfigUpdate {
			stagingTarget = domain.TargetID(stagingConfigUpdate.Target)
			stagingTargetAvailability, err = reader.GetTargetAppNamingAvailability(ctx, app.ID(), domain.Staging, stagingTarget)

			if err != nil {
				return "", err
			}
		}

		if err = validate.Struct(validate.Of{
			"production.target": productionTargetAvailability.Error(),
			"staging.target":    stagingTargetAvailability.Error(),
		}); err != nil {
			return "", err
		}

		if vcsPatch, isSet := cmd.VCS.TryGet(); isSet {
			if vcsUpdate, hasValue := vcsPatch.TryGet(); hasValue {
				// Take the existing vcs as a reference so the token is not modified if not provided at all
				vcs := app.VCS().Get(domain.NewVCSConfig(url))
				vcs.HasUrl(url)

				if tokenPatch, isSet := vcsUpdate.Token.TryGet(); isSet {
					if token, hasValue := tokenPatch.TryGet(); hasValue {
						vcs.Authenticated(token)
					} else {
						vcs.Public()
					}
				}

				app.UseVersionControl(vcs)
			} else {
				app.RemoveVersionControl()
			}
		}

		if hasProductionConfigUpdate {
			if err = app.WithProductionConfig(create_app.BuildEnvironmentConfig(productionTarget, productionConfigUpdate.Vars), productionTargetAvailability); err != nil {
				return "", err
			}
		}

		if hasStagingConfigUpdate {
			if err = app.WithStagingConfig(create_app.BuildEnvironmentConfig(stagingTarget, stagingConfigUpdate.Vars), stagingTargetAvailability); err != nil {
				return "", err
			}
		}

		if err = writer.Write(ctx, &app); err != nil {
			return "", err
		}

		return cmd.ID, nil
	}
}
