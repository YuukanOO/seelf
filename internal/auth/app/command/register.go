package command

import (
	"context"

	"github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/pkg/validation"
	"github.com/YuukanOO/seelf/pkg/validation/strings"
)

type RegisterCommand struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func Register(
	reader domain.UsersReader,
	writer domain.UsersWriter,
	hasher domain.PasswordHasher,
	generator domain.KeyGenerator,
) func(context.Context, RegisterCommand) (string, error) {
	return func(ctx context.Context, cmd RegisterCommand) (string, error) {
		var email domain.Email

		if err := validation.Check(validation.Of{
			"email":    validation.Value(cmd.Email, &email, domain.EmailFrom),
			"password": validation.Is(cmd.Password, strings.Required),
		}); err != nil {
			return "", err
		}

		uniqueEmail, err := reader.IsEmailUnique(ctx, email)

		if err != nil {
			return "", err
		}

		password, err := hasher.Hash(cmd.Password)

		if err != nil {
			return "", err
		}

		key, err := generator.Generate()

		if err != nil {
			return "", err
		}

		user := domain.NewUser(uniqueEmail, password, key)

		if err := writer.Write(ctx, &user); err != nil {
			return "", err
		}

		return string(user.ID()), nil
	}
}
