package domain

import (
	"context"
	"time"

	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/event"
	"github.com/YuukanOO/seelf/pkg/id"
	"github.com/YuukanOO/seelf/pkg/storage"
)

var (
	ErrEmailAlreadyTaken      = apperr.New("email_already_taken")
	ErrInvalidEmailOrPassword = apperr.New("invalid_email_or_password")
)

type (
	// VALUE OBJECTS

	UserID       string
	PasswordHash string
	APIKey       string
	UniqueEmail  Email

	// ENTITY

	User struct {
		event.Emitter

		id           UserID
		password     PasswordHash
		email        UniqueEmail
		key          APIKey
		registeredAt time.Time
	}

	// RELATED SERVICES

	PasswordHasher interface {
		Hash(string) (PasswordHash, error)
		Compare(string, PasswordHash) error
	}

	KeyGenerator interface {
		Generate() (APIKey, error)
	}

	UsersReader interface {
		GetIDFromAPIKey(context.Context, APIKey) (UserID, error)
		IsEmailUnique(context.Context, Email) (UniqueEmail, error)
		IsEmailUniqueForUser(context.Context, UserID, Email) (UniqueEmail, error)
		GetByEmail(context.Context, Email) (User, error)
		GetByID(context.Context, UserID) (User, error)
	}

	UsersWriter interface {
		Write(context.Context, ...*User) error
	}

	// EVENTS

	UserRegistered struct {
		ID           UserID
		Email        UniqueEmail
		Password     PasswordHash
		Key          APIKey
		RegisteredAt time.Time
	}

	UserEmailChanged struct {
		ID    UserID
		Email UniqueEmail
	}

	UserPasswordChanged struct {
		ID       UserID
		Password PasswordHash
	}
)

func NewUser(email UniqueEmail, password PasswordHash, key APIKey) (u User) {
	u.apply(UserRegistered{
		ID:           id.New[UserID](),
		Email:        email,
		Password:     password,
		RegisteredAt: time.Now().UTC(),
		Key:          key,
	})

	return u
}

// Recreates a user from a storage driver
func UserFrom(scanner storage.Scanner) (u User, err error) {
	err = scanner.Scan(
		&u.id,
		&u.email,
		&u.password,
		&u.key,
		&u.registeredAt,
	)

	return u, err
}

// Updates the user email
func (u *User) HasEmail(email UniqueEmail) {
	if u.email == email {
		return
	}

	u.apply(UserEmailChanged{
		ID:    u.id,
		Email: email,
	})
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

func (u User) ID() UserID             { return u.id }
func (u User) Password() PasswordHash { return u.password }

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
	}

	event.Store(u, e)
}
