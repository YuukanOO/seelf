package domain_test

import (
	"testing"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

func Test_Credentials(t *testing.T) {
	t.Run("should be instantiable", func(t *testing.T) {
		cred := domain.NewCredentials("user", "pass")

		testutil.Equals(t, "user", cred.Username())
		testutil.Equals(t, "pass", cred.Password())
	})

	t.Run("should be able to change the username", func(t *testing.T) {
		cred := domain.NewCredentials("user", "pass")

		cred.HasUsername("newuser")

		testutil.Equals(t, "newuser", cred.Username())
		testutil.Equals(t, "pass", cred.Password())
	})

	t.Run("should be able to change the password", func(t *testing.T) {
		cred := domain.NewCredentials("user", "pass")

		cred.HasPassword("newpass")

		testutil.Equals(t, "user", cred.Username())
		testutil.Equals(t, "newpass", cred.Password())
	})
}
