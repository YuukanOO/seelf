//go:build !release

package fixture

import (
	"github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/pkg/id"
	"github.com/YuukanOO/seelf/pkg/must"
)

type (
	userOption struct {
		email        domain.Email
		passwordHash domain.PasswordHash
		apiKey       domain.APIKey
	}

	UserOptionBuilder func(*userOption)
)

func User(options ...UserOptionBuilder) domain.User {
	opts := userOption{
		email:        "john" + id.New[domain.Email]() + "@doe.com",
		passwordHash: id.New[domain.PasswordHash](),
		apiKey:       id.New[domain.APIKey](),
	}

	for _, o := range options {
		o(&opts)
	}

	return must.Panic(domain.NewUser(
		domain.NewEmailRequirement(opts.email, true),
		opts.passwordHash,
		opts.apiKey,
	))
}

func WithEmail(email domain.Email) UserOptionBuilder {
	return func(o *userOption) {
		o.email = email
	}
}

func WithPasswordHash(passwordHash domain.PasswordHash) UserOptionBuilder {
	return func(o *userOption) {
		o.passwordHash = passwordHash
	}
}

func WithPassword(password string, hasher domain.PasswordHasher) UserOptionBuilder {
	return func(o *userOption) {
		o.passwordHash = must.Panic(hasher.Hash(password))
	}
}

func WithAPIKey(apiKey domain.APIKey) UserOptionBuilder {
	return func(o *userOption) {
		o.apiKey = apiKey
	}
}
