package command

import (
	"context"

	"github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/validation"
	"github.com/YuukanOO/seelf/pkg/validation/strings"
)

type UpdateUserCommand struct {
	ID       string              `json:"-"`
	Email    monad.Maybe[string] `json:"email"`
	Password monad.Maybe[string] `json:"password"`
}

// Update user profile.
func UpdateUser(
	reader domain.UsersReader,
	writer domain.UsersWriter,
	hasher domain.PasswordHasher,
) func(context.Context, UpdateUserCommand) error {
	return func(ctx context.Context, cmd UpdateUserCommand) error {
		var email domain.Email

		if err := validation.Check(validation.Of{
			"email": validation.Maybe(cmd.Email, func(mail string) error {
				return validation.Value(mail, &email, domain.EmailFrom)
			}),
			"password": validation.Maybe(cmd.Password, func(password string) error {
				return validation.Is(password, strings.Required)
			}),
		}); err != nil {
			return err
		}

		user, err := reader.GetByID(ctx, domain.UserID(cmd.ID))

		if err != nil {
			return err
		}

		if cmd.Email.HasValue() {
			uniqueEmail, err := reader.IsEmailUniqueForUser(ctx, user.ID(), email)

			if err != nil {
				return err
			}

			user.HasEmail(uniqueEmail)
		}

		if cmd.Password.HasValue() {
			hash, err := hasher.Hash(cmd.Password.MustGet())

			if err != nil {
				return err
			}

			user.HasPassword(hash)
		}

		return writer.Write(ctx, &user)
	}
}
