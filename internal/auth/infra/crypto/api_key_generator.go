package crypto

import (
	"github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/pkg/crypto"
)

const apiKeyLengthInBytes = 64

type keyGenerator struct{}

func NewKeyGenerator() domain.KeyGenerator {
	return &keyGenerator{}
}

func (*keyGenerator) Generate() (domain.APIKey, error) {
	return crypto.RandomKey[domain.APIKey](apiKeyLengthInBytes)
}
