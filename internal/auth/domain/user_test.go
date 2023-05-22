package domain_test

import (
	"testing"

	"github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

func Test_User(t *testing.T) {
	t.Run("could be created", func(t *testing.T) {
		email := domain.UniqueEmail("some@email.com")
		password := domain.PasswordHash("someHashedPassword")
		key := domain.APIKey("someapikey")

		u := domain.NewUser(email, password, key)

		testutil.Equals(t, password, u.Password())
		testutil.NotEquals(t, "", u.ID())

		registeredEvent := testutil.EventIs[domain.UserRegistered](t, &u, 0)

		testutil.Equals(t, u.ID(), registeredEvent.ID)
		testutil.Equals(t, email, registeredEvent.Email)
		testutil.Equals(t, u.Password(), registeredEvent.Password)
		testutil.Equals(t, key, registeredEvent.Key)
	})

	t.Run("should be able to change email", func(t *testing.T) {
		u := domain.NewUser("some@email.com", "someHashedPassword", "")

		u.HasEmail("some@email.com") // no change, should not trigger events
		u.HasEmail("newone@email.com")

		testutil.HasNEvents(t, &u, 2)
		evt := testutil.EventIs[domain.UserEmailChanged](t, &u, 1)
		testutil.Equals(t, u.ID(), evt.ID)
		testutil.Equals(t, "newone@email.com", evt.Email)
	})

	t.Run("should be able to change password", func(t *testing.T) {
		u := domain.NewUser("some@email.com", "someHashedPassword", "")

		u.HasPassword("someHashedPassword") // no change, should not trigger events
		u.HasPassword("anotherPassword")

		testutil.HasNEvents(t, &u, 2)
		evt := testutil.EventIs[domain.UserPasswordChanged](t, &u, 1)
		testutil.Equals(t, u.ID(), evt.ID)
		testutil.Equals(t, "anotherPassword", evt.Password)
	})
}
