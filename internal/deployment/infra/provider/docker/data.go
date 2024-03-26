package docker

import (
	"database/sql/driver"
	"strconv"

	"github.com/YuukanOO/seelf/internal/deployment/app/get_target"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/ssh"
	"github.com/YuukanOO/seelf/pkg/storage"
)

const (
	providerKind = "docker"
	defaultUser  = providerKind
	defaultPort  = 22
)

// Docker provider config stored in a target to allow remote deployments.
type Data struct {
	Host       monad.Maybe[ssh.Host]       `json:"host"`
	Port       monad.Maybe[int]            `json:"port"`
	User       monad.Maybe[string]         `json:"user"`
	PrivateKey monad.Maybe[ssh.PrivateKey] `json:"private_key"`
}

func (Data) Kind() string                              { return providerKind }
func (c Data) Fingerprint() string                     { return string(c.Host.Get("")) } // One provider allowed by host
func (c Data) Value() (driver.Value, error)            { return storage.ValueJSON(c) }
func (c Data) Equals(other domain.ProviderConfig) bool { return c == other }

func (c Data) String() string {
	if host, isRemote := c.Host.TryGet(); isRemote {
		return c.User.Get(defaultUser) + "@" + string(host) + ":" + strconv.Itoa(c.Port.Get(defaultPort))
	}

	return "local"
}

// Specific representation of a docker provider config to avoid leak of sensitive data.
type QueryProviderConfig struct {
	Host       monad.Maybe[string]               `json:"host"`
	Port       monad.Maybe[int]                  `json:"port"`
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
