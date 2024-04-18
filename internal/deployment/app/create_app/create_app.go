package create_app

import (
	"context"

	auth "github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/validate"
	"github.com/YuukanOO/seelf/pkg/validate/strings"
)

type (
	// Create a new application.
	Command struct {
		bus.Command[string]

		Name           string                      `json:"name"`
		VersionControl monad.Maybe[VersionControl] `json:"version_control"`
		Production     EnvironmentConfig           `json:"production"`
		Staging        EnvironmentConfig           `json:"staging"`
	}

	EnvironmentConfig struct {
		Target string                                    `json:"target"`
		Vars   monad.Maybe[map[string]map[string]string] `json:"vars"`
	}

	VersionControl struct {
		Url   string              `json:"url"`
		Token monad.Maybe[string] `json:"token"`
	}
)

func (Command) Name_() string { return "deployment.command.create_app" }

func Handler(
	reader domain.AppsReader,
	writer domain.AppsWriter,
) bus.RequestHandler[string, Command] {
	return func(ctx context.Context, cmd Command) (string, error) {
		var (
			appname          domain.AppName
			url              domain.Url
			productionTarget = domain.TargetID(cmd.Production.Target)
			stagingTarget    = domain.TargetID(cmd.Staging.Target)
		)

		if err := validate.Struct(validate.Of{
			"name": validate.Value(cmd.Name, &appname, domain.AppNameFrom),
			"version_control": validate.Maybe(cmd.VersionControl, func(config VersionControl) error {
				return validate.Struct(validate.Of{
					"url":   validate.Value(config.Url, &url, domain.UrlFrom),
					"token": validate.Maybe(config.Token, strings.Required),
				})
			}),
			"production": validate.Struct(validate.Of{
				"target": validate.Field(cmd.Production.Target, strings.Required),
			}),
			"staging": validate.Struct(validate.Of{
				"target": validate.Field(cmd.Staging.Target, strings.Required),
			}),
		}); err != nil {
			return "", err
		}

		productionRequirement, stagingRequirement, err := reader.CheckAppNamingAvailability(
			ctx,
			appname,
			BuildEnvironmentConfig(productionTarget, cmd.Production.Vars),
			BuildEnvironmentConfig(stagingTarget, cmd.Staging.Vars),
		)

		if err != nil {
			return "", err
		}

		// Returns early if the application name is not unique on both targets.
		if err = validate.Struct(validate.Of{
			"production.target": productionRequirement.Error(),
			"staging.target":    stagingRequirement.Error(),
		}); err != nil {
			return "", err
		}

		app, err := domain.NewApp(
			appname,
			productionRequirement,
			stagingRequirement,
			auth.CurrentUser(ctx).MustGet(),
		)

		if err != nil {
			return "", err
		}

		if cmdVCS, isSet := cmd.VersionControl.TryGet(); isSet {
			vcs := domain.NewVersionControl(url)

			if token, isSet := cmdVCS.Token.TryGet(); isSet {
				vcs.Authenticated(token)
			}

			app.UseVersionControl(vcs)
		}

		if err := writer.Write(ctx, &app); err != nil {
			return "", err
		}

		return string(app.ID()), nil
	}
}

// Helper method to build a domain.EnvironmentConfig from a raw command value.
func BuildEnvironmentConfig(target domain.TargetID, env monad.Maybe[map[string]map[string]string]) domain.EnvironmentConfig {
	config := domain.NewEnvironmentConfig(target)

	if vars, hasVars := env.TryGet(); hasVars {
		config.HasEnvironmentVariables(domain.ServicesEnvFrom(vars))
	}

	return config
}
