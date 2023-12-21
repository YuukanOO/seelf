package login

import (
	"context"
	"errors"

	"github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/validation"
	"github.com/YuukanOO/seelf/pkg/validation/strings"
)

// Log the user in.
type Command struct {
	bus.Command[string]

	Email    string `json:"email"`
	Password string `json:"password"`
}

func (Command) Name_() string { return "auth.command.login" }

func Handler(
	reader domain.UsersReader,
	hasher domain.PasswordHasher,
) bus.RequestHandler[string, Command] {
	return func(ctx context.Context, cmd Command) (string, error) {
		var email domain.Email

		if err := validation.Check(validation.Of{
			"email":    validation.Value(cmd.Email, &email, domain.EmailFrom),
			"password": validation.Is(cmd.Password, strings.Required),
		}); err != nil {
			return "", err
		}

		user, err := reader.GetByEmail(ctx, email)

		if err != nil {
			if errors.Is(err, apperr.ErrNotFound) {
				return "", validation.WrapIfAppErr(domain.ErrInvalidEmailOrPassword, "email", "password")
			}

			return "", err
		}

		if err = hasher.Compare(cmd.Password, user.Password()); err != nil {
			return "", validation.WrapIfAppErr(domain.ErrInvalidEmailOrPassword, "email", "password")
		}

		return string(user.ID()), nil
	}
}
