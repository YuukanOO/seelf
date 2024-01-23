package domain_test

import (
	"testing"

	"github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/pkg/must"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

func Test_User(t *testing.T) {
	t.Run("should fail if the email is not available", func(t *testing.T) {
		_, err := domain.NewUser("an@email.com", "password", "", false)
		testutil.Equals(t, domain.ErrEmailAlreadyTaken, err)
	})

	t.Run("could be created", func(t *testing.T) {
		var (
			email    domain.Email        = "some@email.com"
			password domain.PasswordHash = "someHashedPassword"
			key      domain.APIKey       = "someapikey"
		)

		u, err := domain.NewUser(email, password, key, true)

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
		u := must.Panic(domain.NewUser("some@email.com", "someHashedPassword", "", true))

		err := u.HasEmail("one@email.com", false)
		testutil.Equals(t, domain.ErrEmailAlreadyTaken, err)
	})

	t.Run("should be able to change email", func(t *testing.T) {
		u := must.Panic(domain.NewUser("some@email.com", "someHashedPassword", "", true))

		u.HasEmail("some@email.com", true) // no change, should not trigger events
		u.HasEmail("newone@email.com", true)

		testutil.HasNEvents(t, &u, 2)
		evt := testutil.EventIs[domain.UserEmailChanged](t, &u, 1)
		testutil.Equals(t, u.ID(), evt.ID)
		testutil.Equals(t, "newone@email.com", evt.Email)
	})

	t.Run("should be able to change password", func(t *testing.T) {
		u := must.Panic(domain.NewUser("some@email.com", "someHashedPassword", "", true))

		u.HasPassword("someHashedPassword") // no change, should not trigger events
		u.HasPassword("anotherPassword")

		testutil.HasNEvents(t, &u, 2)
		evt := testutil.EventIs[domain.UserPasswordChanged](t, &u, 1)
		testutil.Equals(t, u.ID(), evt.ID)
		testutil.Equals(t, "anotherPassword", evt.Password)
	})
}
