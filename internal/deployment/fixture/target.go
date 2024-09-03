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
		url      domain.Url
		provider domain.ProviderConfig
		uid      auth.UserID
	}

	TargetOptionBuilder func(*targetOption)
)

func Target(options ...TargetOptionBuilder) domain.Target {
	opts := targetOption{
		name:     id.New[string](),
		url:      must.Panic(domain.UrlFrom("http://" + id.New[string]() + ".com")),
		provider: ProviderConfig(),
		uid:      id.New[auth.UserID](),
	}

	for _, o := range options {
		o(&opts)
	}

	return must.Panic(domain.NewTarget(opts.name,
		domain.NewTargetUrlRequirement(opts.url, true),
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

func WithTargetUrl(url domain.Url) TargetOptionBuilder {
	return func(opts *targetOption) {
		opts.url = url
	}
}

func WithProviderConfig(config domain.ProviderConfig) TargetOptionBuilder {
	return func(opts *targetOption) {
		opts.provider = config
	}
}

type (
	providerConfig struct {
		Data         string
		Fingerprint_ string
	}

	ProviderConfigBuilder func(*providerConfig)
)

func ProviderConfig(options ...ProviderConfigBuilder) domain.ProviderConfig {
	config := providerConfig{
		Data:         id.New[string](),
		Fingerprint_: id.New[string](),
	}

	for _, o := range options {
		o(&config)
	}

	return config
}

func WithFingerprint(fingerprint string) ProviderConfigBuilder {
	return func(config *providerConfig) {
		config.Fingerprint_ = fingerprint
	}
}

func (d providerConfig) Kind() string                 { return "test" }
func (d providerConfig) Fingerprint() string          { return d.Fingerprint_ }
func (d providerConfig) String() string               { return d.Fingerprint_ }
func (d providerConfig) Value() (driver.Value, error) { return storage.ValueJSON(d) }

func (d providerConfig) Equals(other domain.ProviderConfig) bool {
	return d == other
}

func init() {
	domain.ProviderConfigTypes.Register(providerConfig{}, func(s string) (domain.ProviderConfig, error) {
		return storage.UnmarshalJSON[providerConfig](s)
	})
}
