package domain_test

import (
	"testing"

	"github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/pkg/must"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

func Test_User(t *testing.T) {
	t.Run("should fail if the email is not available", func(t *testing.T) {
		_, err := domain.NewUser(domain.NewEmailRequirement("an@email.com", false), "password", "apikey")
		testutil.Equals(t, domain.ErrEmailAlreadyTaken, err)
	})

	t.Run("could be created", func(t *testing.T) {
		var (
			email    domain.Email        = "some@email.com"
			password domain.PasswordHash = "someHashedPassword"
			key      domain.APIKey       = "someapikey"
		)

		u, err := domain.NewUser(domain.NewEmailRequirement(email, true), password, key)

		testutil.IsNil(t, err)
		testutil.Equals(t, password, u.Password())
		testutil.NotEquals(t, "", u.ID())

		registeredEvent := testutil.EventIs[domain.UserRegistered](t, &u, 0)

		testutil.Equals(t, u.ID(), registeredEvent.ID)
		testutil.Equals(t, email, registeredEvent.Email)
		testutil.Equals(t, u.Password(), registeredEvent.Password)
		testutil.Equals(t, key, registeredEvent.Key)
	})

	t.Run("should fail if trying to change for a non available email", func(t *testing.T) {
		u := must.Panic(domain.NewUser(domain.NewEmailRequirement("some@email.com", true), "someHashedPassword", "apikey"))

		err := u.HasEmail(domain.NewEmailRequirement("one@email.com", false))
		testutil.Equals(t, domain.ErrEmailAlreadyTaken, err)
	})

	t.Run("should be able to change email", func(t *testing.T) {
		u := must.Panic(domain.NewUser(domain.NewEmailRequirement("some@email.com", true), "someHashedPassword", "apikey"))

		u.HasEmail(domain.NewEmailRequirement("some@email.com", true)) // no change, should not trigger events
		u.HasEmail(domain.NewEmailRequirement("newone@email.com", true))

		testutil.HasNEvents(t, &u, 2)
		evt := testutil.EventIs[domain.UserEmailChanged](t, &u, 1)
		testutil.Equals(t, u.ID(), evt.ID)
		testutil.Equals(t, "newone@email.com", evt.Email)
	})

	t.Run("should be able to change password", func(t *testing.T) {
		u := must.Panic(domain.NewUser(domain.NewEmailRequirement("some@email.com", true), "someHashedPassword", "apikey"))

		u.HasPassword("someHashedPassword") // no change, should not trigger events
		u.HasPassword("anotherPassword")

		testutil.HasNEvents(t, &u, 2)
		evt := testutil.EventIs[domain.UserPasswordChanged](t, &u, 1)
		testutil.Equals(t, u.ID(), evt.ID)
		testutil.Equals(t, "anotherPassword", evt.Password)
	})

	t.Run("should be able to change API key", func(t *testing.T) {
		u := must.Panic(domain.NewUser(domain.NewEmailRequirement("some@email.com", true), "someHashedPassword", "apikey"))

		u.HasAPIKey("apikey") // no change, should not trigger events
		u.HasAPIKey("anotherKey")

		testutil.HasNEvents(t, &u, 2)
		evt := testutil.EventIs[domain.UserAPIKeyChanged](t, &u, 1)
		testutil.Equals(t, u.ID(), evt.ID)
		testutil.Equals(t, "anotherKey", evt.Key)
	})
}
