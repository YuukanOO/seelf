package crypto_test

import (
	"testing"

	"github.com/YuukanOO/seelf/internal/auth/infra/crypto"
	"github.com/YuukanOO/seelf/pkg/assert"
)

func Test_KeyGenerator(t *testing.T) {
	t.Run("should generate an API key", func(t *testing.T) {
		generator := crypto.NewKeyGenerator()
		key, err := generator.Generate()

		assert.Nil(t, err)
		assert.HasNRunes(t, 64, key)
	})
}
