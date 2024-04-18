package create_first_account

import (
	"context"
	"errors"

	"github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/validate"
	"github.com/YuukanOO/seelf/pkg/validate/strings"
)

var ErrAdminAccountRequired = errors.New(`seelf requires a default user to be created but your database looks empty.
	Please set the SEELF_ADMIN_EMAIL and SEELF_ADMIN_PASSWORD environment variables and relaunch the command, for example:

	$ SEELF_ADMIN_EMAIL=admin@example.com SEELF_ADMIN_PASSWORD=admin seelf serve

	Please note this is a one time only action`)

// Creates the first user account if no one exists yet.
// If an account already exists, its id will be returned.
type Command struct {
	bus.Command[string]

	Email    string `json:"email"`
	Password string `json:"password"`
}

func (Command) Name_() string { return "auth.command.create_first_account" }

func Handler(
	reader domain.UsersReader,
	writer domain.UsersWriter,
	hasher domain.PasswordHasher,
	generator domain.KeyGenerator,
) bus.RequestHandler[string, Command] {
	return func(ctx context.Context, cmd Command) (string, error) {
		user, err := reader.GetAdminUser(ctx)

		if err != nil && !errors.Is(err, apperr.ErrNotFound) {
			return "", err
		}

		// Nothing to do if there is already a user.
		if err == nil {
			return string(user.ID()), nil
		}

		// Some are empty, that's an error!
		if strings.Required(cmd.Email) != nil || strings.Required(cmd.Password) != nil {
			return "", ErrAdminAccountRequired
		}

		var email domain.Email

		if err := validate.Struct(validate.Of{
			"email":    validate.Value(cmd.Email, &email, domain.EmailFrom),
			"password": validate.Field(cmd.Password, strings.Required),
		}); err != nil {
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

		// Here the email uniqueness is guaranteed to be true since we check for the user counts above.
		user, err = domain.NewUser(domain.NewEmailRequirement(email, true), password, key)

		if err != nil {
			return "", err
		}

		if err = writer.Write(ctx, &user); err != nil {
			return "", err
		}

		return string(user.ID()), nil
	}
}
