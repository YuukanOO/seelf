package domain

import (
	"context"
	"time"

	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/bus"
	"github.com/YuukanOO/seelf/pkg/event"
	"github.com/YuukanOO/seelf/pkg/id"
	"github.com/YuukanOO/seelf/pkg/storage"
)

var (
	ErrEmailAlreadyTaken      = apperr.New("email_already_taken")
	ErrInvalidEmailOrPassword = apperr.New("invalid_email_or_password")
)

type (
	UserID       string
	PasswordHash string
	APIKey       string

	User struct {
		event.Emitter

		id           UserID
		password     PasswordHash
		email        Email
		key          APIKey
		registeredAt time.Time
	}

	PasswordHasher interface {
		Hash(string) (PasswordHash, error)
		Compare(string, PasswordHash) error
	}

	KeyGenerator interface {
		Generate() (APIKey, error)
	}

	UsersReader interface {
		GetAdminUser(context.Context) (User, error)
		CheckEmailAvailability(context.Context, Email, ...UserID) (EmailRequirement, error)
		GetByEmail(context.Context, Email) (User, error)
		GetByID(context.Context, UserID) (User, error)
	}

	UsersWriter interface {
		Write(context.Context, ...*User) error
	}

	UserRegistered struct {
		bus.Notification

		ID           UserID
		Email        Email
		Password     PasswordHash
		Key          APIKey
		RegisteredAt time.Time
	}

	UserEmailChanged struct {
		bus.Notification

		ID    UserID
		Email Email
	}

	UserPasswordChanged struct {
		bus.Notification

		ID       UserID
		Password PasswordHash
	}

	UserAPIKeyChanged struct {
		bus.Notification

		ID  UserID
		Key APIKey
	}
)

func (UserRegistered) Name_() string      { return "auth.event.user_registered" }
func (UserEmailChanged) Name_() string    { return "auth.event.user_email_changed" }
func (UserPasswordChanged) Name_() string { return "auth.event.user_password_changed" }
func (UserAPIKeyChanged) Name_() string   { return "auth.event.user_api_key_changed" }

func NewUser(emailRequirement EmailRequirement, password PasswordHash, key APIKey) (u User, err error) {
	email, err := emailRequirement.Met()

	if err != nil {
		return u, err
	}

	u.apply(UserRegistered{
		ID:           id.New[UserID](),
		Email:        email,
		Password:     password,
		RegisteredAt: time.Now().UTC(),
		Key:          key,
	})

	return u, nil
}

// Recreates a user from a storage driver
func UserFrom(scanner storage.Scanner) (u User, err error) {
	var version event.Version

	err = scanner.Scan(
		&u.id,
		&u.email,
		&u.password,
		&u.key,
		&u.registeredAt,
		&version,
	)

	event.Hydrate(&u, version)

	return u, err
}

// Updates the user email
func (u *User) HasEmail(emailRequirement EmailRequirement) error {
	email, err := emailRequirement.Met()

	if err != nil {
		return err
	}

	if u.email == email {
		return nil
	}

	u.apply(UserEmailChanged{
		ID:    u.id,
		Email: email,
	})

	return nil
}

// Updates the user password
func (u *User) HasPassword(password PasswordHash) {
	if u.password == password {
		return
	}

	u.apply(UserPasswordChanged{
		ID:       u.id,
		Password: password,
	})
}

// Updates the user API key
func (u *User) HasAPIKey(key APIKey) {
	if u.key == key {
		return
	}

	u.apply(UserAPIKeyChanged{
		ID:  u.id,
		Key: key,
	})
}

func (u *User) ID() UserID             { return u.id }
func (u *User) Password() PasswordHash { return u.password }

func (u *User) apply(e event.Event) {
	switch evt := e.(type) {
	case UserRegistered:
		u.id = evt.ID
		u.email = evt.Email
		u.password = evt.Password
		u.registeredAt = evt.RegisteredAt
		u.key = evt.Key
	case UserEmailChanged:
		u.email = evt.Email
	case UserPasswordChanged:
		u.password = evt.Password
	case UserAPIKeyChanged:
		u.key = evt.Key
	}

	event.Store(u, e)
}
