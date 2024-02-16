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

		Name       string                 `json:"name"`
		VCS        monad.Maybe[VCSConfig] `json:"vcs"`
		Production EnvironmentConfig      `json:"production"`
		Staging    EnvironmentConfig      `json:"staging"`
	}

	EnvironmentConfig struct {
		Target string                                    `json:"target"`
		Vars   monad.Maybe[map[string]map[string]string] `json:"vars"`
	}

	VCSConfig struct {
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
			productionTarget domain.TargetID = domain.TargetID(cmd.Production.Target)
			stagingTarget    domain.TargetID = domain.TargetID(cmd.Staging.Target)
		)

		if err := validate.Struct(validate.Of{
			"name": validate.Value(cmd.Name, &appname, domain.AppNameFrom),
			"vcs": validate.Maybe(cmd.VCS, func(config VCSConfig) error {
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

		availability, err := reader.GetAppNamingAvailability(ctx, appname, productionTarget, stagingTarget)

		if err != nil {
			return "", err
		}

		// Returns early if the application name is not unique on both targets.
		// Convert the AppNaming flag to a user friendly error.
		if err = validate.Struct(validate.Of{
			"production.target": availability.Error(domain.Production),
			"staging.target":    availability.Error(domain.Staging),
		}); err != nil {
			return "", err
		}

		app, err := domain.NewApp(
			appname,
			BuildEnvironmentConfig(productionTarget, cmd.Production.Vars),
			BuildEnvironmentConfig(stagingTarget, cmd.Staging.Vars),
			availability,
			auth.CurrentUser(ctx).MustGet(),
		)

		if err != nil {
			return "", err
		}

		if cmdVCS, isSet := cmd.VCS.TryGet(); isSet {
			vcs := domain.NewVCSConfig(url)

			if token, isSet := cmdVCS.Token.TryGet(); isSet {
				vcs = vcs.Authenticated(token)
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
		config = config.WithEnvironmentVariables(domain.ServicesEnvFrom(vars))
	}

	return config
}
