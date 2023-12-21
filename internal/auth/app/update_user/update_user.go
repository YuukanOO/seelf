package update_user

import (
	"context"

	"github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/validation"
	"github.com/YuukanOO/seelf/pkg/validation/strings"
)

// Update a user profile.
type Command struct {
	bus.Command[string]

	ID       string              `json:"-"`
	Email    monad.Maybe[string] `json:"email"`
	Password monad.Maybe[string] `json:"password"`
}

func (Command) Name_() string { return "auth.command.update_user" }

func Handler(
	reader domain.UsersReader,
	writer domain.UsersWriter,
	hasher domain.PasswordHasher,
) bus.RequestHandler[string, Command] {
	return func(ctx context.Context, cmd Command) (string, error) {
		var email domain.Email

		if err := validation.Check(validation.Of{
			"email": validation.Maybe(cmd.Email, func(mail string) error {
				return validation.Value(mail, &email, domain.EmailFrom)
			}),
			"password": validation.Maybe(cmd.Password, func(password string) error {
				return validation.Is(password, strings.Required)
			}),
		}); err != nil {
			return "", err
		}

		user, err := reader.GetByID(ctx, domain.UserID(cmd.ID))

		if err != nil {
			return "", err
		}

		if cmd.Email.HasValue() {
			uniqueEmail, err := reader.IsEmailUniqueForUser(ctx, user.ID(), email)

			if err != nil {
				return "", err
			}

			user.HasEmail(uniqueEmail)
		}

		if newPassword, isSet := cmd.Password.TryGet(); isSet {
			hash, err := hasher.Hash(newPassword)

			if err != nil {
				return "", err
			}

			user.HasPassword(hash)
		}

		if err := writer.Write(ctx, &user); err != nil {
			return "", err
		}

		return cmd.ID, nil
	}
}
