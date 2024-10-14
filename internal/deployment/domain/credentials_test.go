package domain_test

import (
	"testing"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/assert"
)

func Test_Credentials(t *testing.T) {
	t.Run("should be instantiable", func(t *testing.T) {
		cred := domain.NewCredentials("user", "pass")

		assert.Equal(t, "user", cred.Username())
		assert.Equal(t, "pass", cred.Password())
	})

	t.Run("should be able to change the username", func(t *testing.T) {
		cred := domain.NewCredentials("user", "pass")

		cred.HasUsername("newuser")

		assert.Equal(t, "newuser", cred.Username())
		assert.Equal(t, "pass", cred.Password())
	})

	t.Run("should be able to change the password", func(t *testing.T) {
		cred := domain.NewCredentials("user", "pass")

		cred.HasPassword("newpass")

		assert.Equal(t, "user", cred.Username())
		assert.Equal(t, "newpass", cred.Password())
	})
}
