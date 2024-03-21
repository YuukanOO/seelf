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
	Url      monad.Maybe[string] `json:"url"`
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
			"url": validate.Maybe(cmd.Url, func(s string) error {
				return validate.Value(s, &targetUrl, domain.UrlFrom)
			}),
		}); err != nil {
			return "", err
		}

		target, err := reader.GetByID(ctx, domain.TargetID(cmd.ID))

		if err != nil {
			return "", err
		}

		if name, isSet := cmd.Name.TryGet(); isSet {
			if err = target.Rename(name); err != nil {
				return "", err
			}
		}

		if cmd.Url.HasValue() {
			// Validate availability of the target url
			availability, err := reader.GetUrlAvailability(ctx, targetUrl, target.ID())

			if err != nil {
				return "", err
			}

			if err = availability.Error(); err != nil {
				return "", validate.Wrap(err, "url")
			}

			if err = target.HasUrl(targetUrl, availability); err != nil {
				return "", err
			}
		}

		if cmd.Provider != nil {
			config, err := provider.Prepare(ctx, cmd.Provider, target.Provider())

			if err != nil {
				return "", err
			}

			availability, err := reader.GetConfigAvailability(ctx, config, target.ID())

			if err != nil {
				return "", err
			}

			if err = availability.Error(); err != nil {
				return "", validate.Wrap(err, config.Kind())
			}

			if err = target.HasProvider(config, availability); err != nil {
				return "", err
			}
		}

		if err = writer.Write(ctx, &target); err != nil {
			return "", err
		}

		return cmd.ID, nil
	}
}
