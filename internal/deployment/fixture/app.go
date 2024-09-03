//go:build !release

package fixture

import (
	auth "github.com/YuukanOO/seelf/internal/auth/domain"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/id"
	"github.com/YuukanOO/seelf/pkg/must"
)

type (
	appOption struct {
		name       domain.AppName
		production domain.EnvironmentConfig
		staging    domain.EnvironmentConfig
		createdBy  auth.UserID
	}

	AppOptionBuilder func(*appOption)
)

func App(options ...AppOptionBuilder) domain.App {
	opts := appOption{
		name:       id.New[domain.AppName](),
		production: domain.NewEnvironmentConfig(id.New[domain.TargetID]()),
		staging:    domain.NewEnvironmentConfig(id.New[domain.TargetID]()),
		createdBy:  id.New[auth.UserID](),
	}

	for _, o := range options {
		o(&opts)
	}

	return must.Panic(domain.NewApp(opts.name,
		domain.NewEnvironmentConfigRequirement(opts.production, true, true),
		domain.NewEnvironmentConfigRequirement(opts.staging, true, true),
		opts.createdBy,
	))
}

func WithAppName(name domain.AppName) AppOptionBuilder {
	return func(o *appOption) {
		o.name = name
	}
}

func WithAppCreatedBy(uid auth.UserID) AppOptionBuilder {
	return func(o *appOption) {
		o.createdBy = uid
	}
}

func WithProductionConfig(production domain.EnvironmentConfig) AppOptionBuilder {
	return func(o *appOption) {
		o.production = production
	}
}

func WithEnvironmentConfig(production, staging domain.EnvironmentConfig) AppOptionBuilder {
	return func(o *appOption) {
		o.production = production
		o.staging = staging
	}
}
