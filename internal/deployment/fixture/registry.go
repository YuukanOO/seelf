//go:build !release

package fixture

import (
	auth "github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/id"
	"github.com/YuukanOO/seelf/pkg/must"
)

type (
	registryOption struct {
		name string
		url  domain.Url
		uid  auth.UserID
	}

	RegistryOptionBuilder func(*registryOption)
)

func Registry(options ...RegistryOptionBuilder) domain.Registry {
	opts := registryOption{
		name: id.New[string](),
		url:  must.Panic(domain.UrlFrom("http://" + id.New[string]() + ".com")),
		uid:  id.New[auth.UserID](),
	}

	for _, o := range options {
		o(&opts)
	}

	return must.Panic(domain.NewRegistry(opts.name, domain.NewRegistryUrlRequirement(opts.url, true), opts.uid))
}

func WithRegistryName(name string) RegistryOptionBuilder {
	return func(o *registryOption) {
		o.name = name
	}
}

func WithRegistryCreatedBy(uid auth.UserID) RegistryOptionBuilder {
	return func(o *registryOption) {
		o.uid = uid
	}
}

func WithUrl(url domain.Url) RegistryOptionBuilder {
	return func(o *registryOption) {
		o.url = url
	}
}
