package ssh

import (
	"net"
	"regexp"

	"github.com/YuukanOO/seelf/pkg/apperr"
)

var (
	ErrInvalidHost = apperr.New("invalid_host")

	hostRegex = regexp.MustCompile(`^([a-zA-Z0-9]{1}[a-zA-Z0-9-]{0,62}){1}(\.[a-zA-Z0-9]{1}[a-zA-Z0-9-]{0,62})*?$`)
)

type Host string

// Parses a raw value as a host. Allow ipv4 & ipv6 addresses and domain names without port number.
func ParseHost(value string) (Host, error) {
	if net.ParseIP(value) == nil && !hostRegex.MatchString(value) {
		return "", ErrInvalidHost
	}

	return Host(value), nil
}

func (h Host) String() string { return string(h) }
