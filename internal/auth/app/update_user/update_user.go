package update_user

import (
	"context"

	"github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/validate"
	"github.com/YuukanOO/seelf/pkg/validate/strings"
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

		if err := validate.Struct(validate.Of{
			"email": validate.Maybe(cmd.Email, func(mail string) error {
				return validate.Value(mail, &email, domain.EmailFrom)
			}),
			"password": validate.Maybe(cmd.Password, func(password string) error {
				return validate.Field(password, strings.Required)
			}),
		}); err != nil {
			return "", err
		}

		user, err := reader.GetByID(ctx, domain.UserID(cmd.ID))

		if err != nil {
			return "", err
		}

		if cmd.Email.HasValue() {
			emailRequirement, err := reader.CheckEmailAvailability(ctx, email, user.ID())

			if err != nil {
				return "", err
			}

			if err = user.HasEmail(emailRequirement); err != nil {
				return "", validate.Wrap(err, "email")
			}
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
