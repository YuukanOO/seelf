package command

import (
	"context"
	"errors"

	"github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/validation"
	"github.com/YuukanOO/seelf/pkg/validation/strings"
)

type LoginCommand struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func Login(
	reader domain.UsersReader,
	hasher domain.PasswordHasher,
) func(context.Context, LoginCommand) (string, error) {
	return func(ctx context.Context, cmd LoginCommand) (string, error) {
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
