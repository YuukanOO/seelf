package crypto_test

import (
	"testing"

	"github.com/YuukanOO/seelf/internal/auth/infra/crypto"
	"github.com/YuukanOO/seelf/pkg/testutil"
	"golang.org/x/crypto/bcrypt"
)

var hasher = crypto.NewBCryptHasher()

func Test_BCryptHasher_ShouldHashPassword(t *testing.T) {
	t.Run("should hash password", func(t *testing.T) {
		hash, err := hasher.Hash("mysecretpassword")
		testutil.IsNil(t, err)
		testutil.HasNChars(t, 60, hash)
	})

	t.Run("should compare password", func(t *testing.T) {
		hash, _ := hasher.Hash("mysecretpassword")
		err := hasher.Compare("mysecretpassword", hash)
		testutil.IsNil(t, err)

		err = hasher.Compare("anothersecretpassword", hash)
		testutil.IsTrue(t, err == bcrypt.ErrMismatchedHashAndPassword)
	})
}
