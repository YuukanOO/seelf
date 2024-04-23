package refresh_api_key

import (
	"context"

	"github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/pkg/bus"
)

type Command struct {
	bus.Command[string]

	ID string `json:"-"`
}

func (Command) Name_() string { return "auth.command.refresh_api_key" }

func Handler(
	reader domain.UsersReader,
	writer domain.UsersWriter,
	generator domain.KeyGenerator,
) bus.RequestHandler[string, Command] {
	return func(ctx context.Context, cmd Command) (string, error) {
		user, err := reader.GetByID(ctx, domain.UserID(cmd.ID))

		if err != nil {
			return "", err
		}

		key, err := generator.Generate()

		if err != nil {
			return "", err
		}

		user.HasAPIKey(key)

		if err = writer.Write(ctx, &user); err != nil {
			return "", err
		}

		return string(key), nil
	}
}
