package update_registry

import (
	"context"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/validate"
	"github.com/YuukanOO/seelf/pkg/validate/strings"
)

type (
	Command struct {
		bus.Command[string]

		ID          string                   `json:"id"`
		Name        monad.Maybe[string]      `json:"name"`
		Url         monad.Maybe[string]      `json:"url"`
		Credentials monad.Patch[Credentials] `json:"credentials"`
	}

	Credentials struct {
		Username string              `json:"username"`
		Password monad.Maybe[string] `json:"password"` // Not set if the user wants to keep the current password.
	}
)

func (Command) Name_() string { return "deployment.command.update_registry" }

func Handler(
	reader domain.RegistriesReader,
	writer domain.RegistriesWriter,
) bus.RequestHandler[string, Command] {
	return func(ctx context.Context, cmd Command) (string, error) {
		var url domain.Url

		if err := validate.Struct(validate.Of{
			"name": validate.Maybe(cmd.Name, strings.Required),
			"url": validate.Maybe(cmd.Url, func(u string) error {
				return validate.Value(u, &url, domain.UrlFrom)
			}),
			"credentials": validate.Patch(cmd.Credentials, func(creds Credentials) error {
				return validate.Struct(validate.Of{
					"username": validate.Field(creds.Username, strings.Required),
					"password": validate.Maybe(creds.Password, strings.Required),
				})
			}),
		}); err != nil {
			return "", err
		}

		registry, err := reader.GetByID(ctx, domain.RegistryID(cmd.ID))

		if err != nil {
			return "", err
		}

		if name, isSet := cmd.Name.TryGet(); isSet {
			registry.Rename(name)
		}

		var urlRequirement domain.RegistryUrlRequirement

		if cmd.Url.HasValue() {
			urlRequirement, err = reader.CheckUrlAvailability(ctx, url, registry.ID())

			if err != nil {
				return "", err
			}

			if err = urlRequirement.Error(); err != nil {
				return "", validate.Wrap(err, "url")
			}

			if err = registry.HasUrl(urlRequirement); err != nil {
				return "", err
			}
		}

		if credentialsPatch, isSet := cmd.Credentials.TryGet(); isSet {
			if credentialsUpdate, hasValue := credentialsPatch.TryGet(); hasValue {
				credentials := registry.Credentials().Get(domain.NewCredentials(credentialsUpdate.Username, credentialsUpdate.Password.Get("")))
				credentials.HasUsername(credentialsUpdate.Username)

				if newPassword, isSet := credentialsUpdate.Password.TryGet(); isSet {
					credentials.HasPassword(newPassword)
				}

				registry.UseAuthentication(credentials)
			} else {
				registry.RemoveAuthentication()
			}
		}

		if err = writer.Write(ctx, &registry); err != nil {
			return "", err
		}

		return cmd.ID, nil
	}
}
