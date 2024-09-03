package crypto_test

import (
	"testing"

	"github.com/YuukanOO/seelf/internal/auth/infra/crypto"
	"github.com/YuukanOO/seelf/pkg/assert"
	"golang.org/x/crypto/bcrypt"
)

func Test_BCryptHasher(t *testing.T) {
	hasher := crypto.NewBCryptHasher()

	t.Run("should hash password", func(t *testing.T) {
		hash, err := hasher.Hash("mysecretpassword")
		assert.Nil(t, err)
		assert.HasNRunes(t, 60, hash)
	})

	t.Run("should compare password", func(t *testing.T) {
		hash, _ := hasher.Hash("mysecretpassword")
		err := hasher.Compare("mysecretpassword", hash)
		assert.Nil(t, err)

		err = hasher.Compare("anothersecretpassword", hash)
		assert.ErrorIs(t, bcrypt.ErrMismatchedHashAndPassword, err)
	})
}
