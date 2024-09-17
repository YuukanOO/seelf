package update_target

import (
	"context"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/validate"
	"github.com/YuukanOO/seelf/pkg/validate/strings"
)

type Command struct {
	bus.Command[string]

	ID       string              `json:"-"`
	Name     monad.Maybe[string] `json:"name"`
	Url      monad.Patch[string] `json:"url"`
	Provider any                 `json:"-"`
}

func (Command) Name_() string { return "deployment.command.update_target" }

func Handler(
	reader domain.TargetsReader,
	writer domain.TargetsWriter,
	provider domain.Provider,
) bus.RequestHandler[string, Command] {
	return func(ctx context.Context, cmd Command) (string, error) {
		var targetUrl domain.Url

		if err := validate.Struct(validate.Of{
			"name": validate.Maybe(cmd.Name, strings.Required),
			"url": validate.Patch(cmd.Url, func(s string) error {
				return validate.Value(s, &targetUrl, domain.UrlFrom)
			}),
		}); err != nil {
			return "", err
		}

		target, err := reader.GetByID(ctx, domain.TargetID(cmd.ID))

		if err != nil {
			return "", err
		}

		// Validate both requirements at once if the value has been updated
		var (
			urlRequirement    domain.TargetUrlRequirement
			configRequirement domain.ProviderConfigRequirement
			configKind        string
		)

		if cmd.Url.HasValue() {
			urlRequirement, err = reader.CheckUrlAvailability(ctx, targetUrl, target.ID())

			if err != nil {
				return "", err
			}
		}

		if cmd.Provider != nil {
			config, err := provider.Prepare(ctx, cmd.Provider, target.Provider())

			if err != nil {
				return "", err
			}

			configKind = config.Kind()
			configRequirement, err = reader.CheckConfigAvailability(ctx, config, target.ID())

			if err != nil {
				return "", err
			}
		}

		if err = validate.Struct(validate.Of{
			"url":      validate.If(cmd.Url.HasValue(), urlRequirement.Error),
			configKind: validate.If(cmd.Provider != nil, configRequirement.Error),
		}); err != nil {
			return "", err
		}

		if name, isSet := cmd.Name.TryGet(); isSet {
			if err = target.Rename(name); err != nil {
				return "", err
			}
		}

		if cmd.Url.IsSet() {
			if cmd.Url.HasValue() {
				err = target.ExposeServicesAutomatically(urlRequirement)
			} else {
				err = target.ExposeServicesManually()
			}

			if err != nil {
				return "", err
			}
		}

		if cmd.Provider != nil {
			if err = target.HasProvider(configRequirement); err != nil {
				return "", err
			}
		}

		if err = writer.Write(ctx, &target); err != nil {
			return "", err
		}

		return cmd.ID, nil
	}
}
