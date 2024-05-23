package create_registry

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
	Command struct {
		bus.Command[string]

		Name        string                   `json:"name"`
		Url         string                   `json:"url"`
		Credentials monad.Maybe[Credentials] `json:"credentials"`
	}

	Credentials struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
)

func (Command) Name_() string { return "deployment.command.create_registry" }

func Handler(
	reader domain.RegistriesReader,
	writer domain.RegistriesWriter,
) bus.RequestHandler[string, Command] {
	return func(ctx context.Context, cmd Command) (string, error) {
		var url domain.Url

		if err := validate.Struct(validate.Of{
			"name": validate.Field(cmd.Name, strings.Required),
			"url":  validate.Value(cmd.Url, &url, domain.UrlFrom),
			"credentials": validate.Maybe(cmd.Credentials, func(creds Credentials) error {
				return validate.Struct(validate.Of{
					"username": validate.Field(creds.Username, strings.Required),
					"password": validate.Field(creds.Password, strings.Required),
				})
			}),
		}); err != nil {
			return "", err
		}

		urlRequirement, err := reader.CheckUrlAvailability(ctx, url)

		if err != nil {
			return "", err
		}

		if err = urlRequirement.Error(); err != nil {
			return "", validate.Wrap(err, "url")
		}

		registry, err := domain.NewRegistry(cmd.Name, urlRequirement, auth.CurrentUser(ctx).MustGet())

		if err != nil {
			return "", err
		}

		if credentials, hasCredentials := cmd.Credentials.TryGet(); hasCredentials {
			registry.UseAuthentication(domain.NewCredentials(credentials.Username, credentials.Password))
		}

		if err = writer.Write(ctx, &registry); err != nil {
			return "", err
		}

		return string(registry.ID()), nil
	}
}
