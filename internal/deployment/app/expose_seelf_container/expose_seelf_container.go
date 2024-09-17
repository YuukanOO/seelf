package expose_seelf_container

import (
	"context"
	"errors"

	auth "github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/validate"
	"github.com/YuukanOO/seelf/pkg/validate/strings"
)

type (
	// Expose seelf container using a local target configuration.
	// If a local target already exists, it will not be modified but only use
	// to attach the container to it.
	Command struct {
		bus.Command[bus.UnitType]

		Container string `json:"container"`
		Url       string `json:"url"`
	}

	LocalProvider interface {
		domain.Provider

		PrepareLocal(context.Context) (domain.ProviderConfig, error) // Prepare a local provider configuration
		Expose(context.Context, domain.Target, string) error         // Attach the given container name to the given target
	}
)

func (Command) Name_() string { return "deployment.command.expose_seelf_container" }

func Handler(
	reader domain.TargetsReader,
	writer domain.TargetsWriter,
	provider LocalProvider,
) bus.RequestHandler[bus.UnitType, Command] {
	return func(ctx context.Context, cmd Command) (bus.UnitType, error) {
		var url domain.Url

		if err := validate.Struct(validate.Of{
			"container": validate.Field(cmd.Container, strings.Required),
			"url":       validate.Value(cmd.Url, &url, domain.UrlFrom),
		}); err != nil {
			return bus.Unit, err
		}

		target, err := reader.GetLocalTarget(ctx)

		if err != nil && !errors.Is(err, apperr.ErrNotFound) {
			return bus.Unit, err
		}

		// Target already exists, just attach the container to the appropriate network
		if err == nil {
			return bus.Unit, provider.Expose(ctx, target, cmd.Container)
		}

		conf, err := provider.PrepareLocal(ctx)

		if err != nil {
			return bus.Unit, err
		}

		urlRequirement, err := reader.CheckUrlAvailability(ctx, url)

		if err != nil {
			return bus.Unit, err
		}

		configRequirement, err := reader.CheckConfigAvailability(ctx, conf)

		if err != nil {
			return bus.Unit, err
		}

		target, err = domain.NewTarget("local",
			configRequirement,
			auth.CurrentUser(ctx).MustGet(),
		)

		if err != nil {
			return bus.Unit, err
		}

		if err = target.ExposeServicesAutomatically(urlRequirement); err != nil {
			return bus.Unit, err
		}

		assigned, err := provider.Setup(ctx, target)

		target.Configured(target.CurrentVersion(), assigned, err)

		if err := writer.Write(ctx, &target); err != nil {
			return bus.Unit, err
		}

		return bus.Unit, provider.Expose(ctx, target, cmd.Container)
	}
}
