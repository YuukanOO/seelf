package delete_registry

import (
	"context"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/bus"
)

type Command struct {
	bus.Command[bus.UnitType]

	ID string `json:"id"`
}

func (Command) Name_() string { return "deployment.command.delete_registry" }

func Handler(
	reader domain.RegistriesReader,
	writer domain.RegistriesWriter,
) bus.RequestHandler[bus.UnitType, Command] {
	return func(ctx context.Context, cmd Command) (bus.UnitType, error) {
		registry, err := reader.GetByID(ctx, domain.RegistryID(cmd.ID))

		if err != nil {
			return bus.Unit, err
		}

		registry.Delete()

		return bus.Unit, writer.Write(ctx, &registry)
	}
}
