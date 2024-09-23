//go:build !release

package fixture

import (
	"database/sql/driver"

	auth "github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/id"
	"github.com/YuukanOO/seelf/pkg/must"
	"github.com/YuukanOO/seelf/pkg/storage"
)

type (
	targetOption struct {
		name     string
		provider domain.ProviderConfig
		uid      auth.UserID
	}

	TargetOptionBuilder func(*targetOption)
)

func Target(options ...TargetOptionBuilder) domain.Target {
	opts := targetOption{
		name:     id.New[string](),
		provider: ProviderConfig(),
		uid:      id.New[auth.UserID](),
	}

	for _, o := range options {
		o(&opts)
	}

	return must.Panic(domain.NewTarget(opts.name,
		domain.NewProviderConfigRequirement(opts.provider, true),
		opts.uid))
}

func WithTargetName(name string) TargetOptionBuilder {
	return func(opts *targetOption) {
		opts.name = name
	}
}

func WithTargetCreatedBy(uid auth.UserID) TargetOptionBuilder {
	return func(opts *targetOption) {
		opts.uid = uid
	}
}

func WithProviderConfig(config domain.ProviderConfig) TargetOptionBuilder {
	return func(opts *targetOption) {
		opts.provider = config
	}
}

type (
	providerConfig struct {
		Kind_        string
		Data         string
		Fingerprint_ string
	}

	ProviderConfigBuilder func(*providerConfig)
)

func ProviderConfig(options ...ProviderConfigBuilder) (result domain.ProviderConfig) {
	config := providerConfig{
		Data:         id.New[string](),
		Kind_:        id.New[string](),
		Fingerprint_: id.New[string](),
	}

	for _, o := range options {
		o(&config)
	}

	result = config

	// Just ignore the panic due to the multiple registration of same kind
	defer func() {
		_ = recover()
	}()

	domain.ProviderConfigTypes.Register(config, func(s string) (domain.ProviderConfig, error) {
		return storage.UnmarshalJSON[providerConfig](s)
	})

	return
}

func WithFingerprint(fingerprint string) ProviderConfigBuilder {
	return func(config *providerConfig) {
		config.Fingerprint_ = fingerprint
	}
}

func WithKind(kind string) ProviderConfigBuilder {
	return func(config *providerConfig) {
		config.Kind_ = kind
	}
}

func WithData(data string) ProviderConfigBuilder {
	return func(config *providerConfig) {
		config.Data = data
	}
}

func (d providerConfig) Kind() string                 { return d.Kind_ }
func (d providerConfig) Fingerprint() string          { return d.Fingerprint_ }
func (d providerConfig) String() string               { return d.Fingerprint_ }
func (d providerConfig) Value() (driver.Value, error) { return storage.ValueJSON(d) }

func (d providerConfig) Equals(other domain.ProviderConfig) bool {
	return d == other
}
