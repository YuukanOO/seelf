package infra

import (
	"github.com/YuukanOO/seelf/internal/auth/domain"
	"golang.org/x/crypto/bcrypt"
)

type bcryptHasher struct{}

func NewBCryptHasher() domain.PasswordHasher {
	return &bcryptHasher{}
}

func (*bcryptHasher) Hash(value string) (domain.PasswordHash, error) {
	data, err := bcrypt.GenerateFromPassword([]byte(value), bcrypt.DefaultCost)

	if err != nil {
		return "", err
	}

	return domain.PasswordHash(data), nil
}

func (*bcryptHasher) Compare(value string, hash domain.PasswordHash) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(value))
}
