package create_app

import (
	"context"

	auth "github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/validation"
	"github.com/YuukanOO/seelf/pkg/validation/strings"
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

		if err := validation.Check(validation.Of{
			"name": validation.Value(cmd.Name, &appname, domain.AppNameFrom),
			"vcs": validation.Maybe(cmd.VCS, func(config VCSConfig) error {
				return validation.Check(validation.Of{
					"url":   validation.Value(config.Url, &url, domain.UrlFrom),
					"token": validation.Maybe(config.Token, strings.Required),
				})
			}),
			"production": validation.Check(validation.Of{
				"target": validation.Is(cmd.Production.Target, strings.Required),
			}),
			"staging": validation.Check(validation.Of{
				"target": validation.Is(cmd.Staging.Target, strings.Required),
			}),
		}); err != nil {
			return "", err
		}

		requirement, err := reader.GetAppNamingAvailability(ctx, appname, productionTarget, stagingTarget)

		if err != nil {
			return "", err
		}

		// // Returns early if the application name is not unique on both targets.
		// if err = validation.Check(validation.Of{
		// 	"production.target": requirement.Production().Error(),
		// 	"staging.target":    requirement.Staging().Error(),
		// }); err != nil {
		// 	return "", err
		// }

		app, err := domain.NewApp(
			appname,
			BuildEnvironmentConfig(productionTarget, cmd.Production.Vars),
			BuildEnvironmentConfig(stagingTarget, cmd.Staging.Vars),
			auth.CurrentUser(ctx).MustGet(),
			requirement,
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
