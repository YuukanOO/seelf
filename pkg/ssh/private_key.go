package ssh

import (
	"github.com/YuukanOO/seelf/pkg/apperr"
	"golang.org/x/crypto/ssh"
)

var ErrInvalidSSHKey = apperr.New("invalid_ssh_key")

type PrivateKey string

// Parses a raw ssh private key.
func ParsePrivateKey(value string) (PrivateKey, error) {
	if _, err := ssh.ParsePrivateKey([]byte(value)); err != nil {
		return "", ErrInvalidSSHKey
	}

	return PrivateKey(value), nil
}
