package create_first_account

import (
	"context"

	"github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/validation"
	"github.com/YuukanOO/seelf/pkg/validation/strings"
)

// Creates the first user account if no one exists yet.
// If an account has been created, its id is returned.
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
		count, err := reader.GetUsersCount(ctx)

		if err != nil {
			return "", err
		}

		// Nothing to do if there is already a user.
		if count > 0 {
			return "", nil
		}

		// Some are empty, that's an error!
		if strings.Required(cmd.Email) != nil || strings.Required(cmd.Password) != nil {
			return "", domain.ErrAdminAccountRequired
		}

		var email domain.Email

		if err := validation.Check(validation.Of{
			"email":    validation.Value(cmd.Email, &email, domain.EmailFrom),
			"password": validation.Is(cmd.Password, strings.Required),
		}); err != nil {
			return "", err
		}

		// Here this line is not mandatory since we are already checking for the count of users.
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

		if err = writer.Write(ctx, &user); err != nil {
			return "", err
		}

		return string(user.ID()), nil
	}
}
