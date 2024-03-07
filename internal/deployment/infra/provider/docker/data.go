package docker

import (
	"database/sql/driver"
	"net"
	"regexp"

	"github.com/YuukanOO/seelf/internal/deployment/app/get_target"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/storage"
	"golang.org/x/crypto/ssh"
)

const providerKind = "docker"

var (
	ErrInvalidHost   = apperr.New("invalid_host")
	ErrInvalidSSHKey = apperr.New("invalid_ssh_key")

	hostRegex = regexp.MustCompile(`^([a-zA-Z0-9]{1}[a-zA-Z0-9-]{0,62}){1}(\.[a-zA-Z0-9]{1}[a-zA-Z0-9-]{0,62})*?$`)
)

type (
	Host       string
	PrivateKey string

	// Docker provider config stored in a target to allow remote deployments.
	Data struct {
		Host       monad.Maybe[Host]       `json:"host"`
		Port       monad.Maybe[uint]       `json:"port"`
		User       monad.Maybe[string]     `json:"user"`
		PrivateKey monad.Maybe[PrivateKey] `json:"private_key"`
	}
)

func (Data) Kind() string                              { return providerKind }
func (c Data) Fingerprint() string                     { return string(c.Host.Get("")) } // One provider allowed by host
func (c Data) Value() (driver.Value, error)            { return storage.ValueJSON(c) }
func (c Data) Equals(other domain.ProviderConfig) bool { return c == other }

// Parses a raw value as a host. Allow ipv4 & ipv6 addresses and domain names without port number.
func HostFrom(value string) (Host, error) {
	if net.ParseIP(value) == nil && !hostRegex.MatchString(value) {
		return "", ErrInvalidHost
	}

	return Host(value), nil
}

// Parses a raw ssh private key.
func PrivateKeyFrom(value string) (PrivateKey, error) {
	if _, err := ssh.ParsePrivateKey([]byte(value)); err != nil {
		return "", ErrInvalidSSHKey
	}

	return PrivateKey(value), nil
}

// Specific representation of a docker provider config to avoid leak of sensitive data.
type QueryProviderConfig struct {
	Host       monad.Maybe[string]               `json:"host"`
	Port       monad.Maybe[uint]                 `json:"port"`
	User       monad.Maybe[string]               `json:"user"`
	PrivateKey monad.Maybe[storage.SecretString] `json:"private_key"`
}

func (QueryProviderConfig) Kind() string { return providerKind }

func init() {
	domain.ProviderConfigTypes.Register(Data{}, func(s string) (domain.ProviderConfig, error) {
		return storage.UnmarshalJSON[Data](s)
	})

	get_target.ProviderConfigTypes.Register(QueryProviderConfig{}, func(s string) (get_target.ProviderConfig, error) {
		return storage.UnmarshalJSON[QueryProviderConfig](s)
	})
}
