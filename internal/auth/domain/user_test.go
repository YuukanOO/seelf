package domain_test

import (
	"testing"

	"github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/internal/auth/fixture"
	"github.com/YuukanOO/seelf/pkg/assert"
)

func Test_User(t *testing.T) {
	t.Run("should fail if the email is not available", func(t *testing.T) {
		_, err := domain.NewUser(domain.NewEmailRequirement("an@email.com", false), "password", "apikey")
		assert.ErrorIs(t, domain.ErrEmailAlreadyTaken, err)
	})

	t.Run("could be created", func(t *testing.T) {
		var (
			email    domain.Email        = "some@email.com"
			password domain.PasswordHash = "someHashedPassword"
			key      domain.APIKey       = "someapikey"
		)

		u, err := domain.NewUser(domain.NewEmailRequirement(email, true), password, key)

		assert.Nil(t, err)
		assert.Equal(t, password, u.Password())
		assert.NotZero(t, u.ID())

		registeredEvent := assert.EventIs[domain.UserRegistered](t, &u, 0)

		assert.Equal(t, domain.UserRegistered{
			ID:           u.ID(),
			Email:        email,
			Password:     password,
			Key:          key,
			RegisteredAt: assert.NotZero(t, registeredEvent.RegisteredAt),
		}, registeredEvent)
	})

	t.Run("should fail if trying to change for a non available email", func(t *testing.T) {
		existingUser := fixture.User()

		err := existingUser.HasEmail(domain.NewEmailRequirement("one@email.com", false))
		assert.ErrorIs(t, domain.ErrEmailAlreadyTaken, err)
	})

	t.Run("should be able to change email", func(t *testing.T) {
		existingUser := fixture.User(fixture.WithEmail("some@email.com"))

		assert.Nil(t, existingUser.HasEmail(domain.NewEmailRequirement("some@email.com", true)))
		assert.Nil(t, existingUser.HasEmail(domain.NewEmailRequirement("newone@email.com", true)))

		assert.HasNEvents(t, 2, &existingUser, "should raise the event once per different email")
		evt := assert.EventIs[domain.UserEmailChanged](t, &existingUser, 1)

		assert.Equal(t, domain.UserEmailChanged{
			ID:    existingUser.ID(),
			Email: "newone@email.com",
		}, evt)
	})

	t.Run("should be able to change password", func(t *testing.T) {
		existingUser := fixture.User(fixture.WithPasswordHash("someHashedPassword"))

		existingUser.HasPassword("someHashedPassword")
		existingUser.HasPassword("anotherPassword")

		assert.HasNEvents(t, 2, &existingUser, "should raise the event once per different password")
		evt := assert.EventIs[domain.UserPasswordChanged](t, &existingUser, 1)

		assert.Equal(t, domain.UserPasswordChanged{
			ID:       existingUser.ID(),
			Password: "anotherPassword",
		}, evt)
	})

	t.Run("should be able to change API key", func(t *testing.T) {
		existingUser := fixture.User(fixture.WithAPIKey("apikey"))

		existingUser.HasAPIKey("apikey")
		existingUser.HasAPIKey("anotherKey")

		assert.HasNEvents(t, 2, &existingUser, "should raise the event once per different API key")
		evt := assert.EventIs[domain.UserAPIKeyChanged](t, &existingUser, 1)

		assert.Equal(t, domain.UserAPIKeyChanged{
			ID:  existingUser.ID(),
			Key: "anotherKey",
		}, evt)
	})
}
