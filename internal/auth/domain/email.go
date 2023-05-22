package domain

import (
	"net/mail"

	"github.com/YuukanOO/seelf/pkg/apperr"
)

var ErrInvalidEmail = apperr.New("invalid_email")

type Email string

// Try to constructs an email from a raw string
func EmailFrom(value string) (Email, error) {
	addr, err := mail.ParseAddress(value)

	if err != nil {
		return "", ErrInvalidEmail
	}

	return Email(addr.Address), nil
}
