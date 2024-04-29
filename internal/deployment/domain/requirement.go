package domain

import "github.com/YuukanOO/seelf/pkg/apperr"

type EnvironmentConfigRequirement struct {
	config       EnvironmentConfig
	targetExists bool
	available    bool
}

func NewEnvironmentConfigRequirement(config EnvironmentConfig, targetExists bool, available bool) EnvironmentConfigRequirement {
	return EnvironmentConfigRequirement{
		config:       config,
		targetExists: targetExists,
		available:    targetExists && available,
	}
}

func (e EnvironmentConfigRequirement) Error() error {
	if !e.targetExists {
		return apperr.ErrNotFound
	}

	if !e.available {
		return ErrAppNameAlreadyTaken
	}

	return nil
}

func (e EnvironmentConfigRequirement) Met() (EnvironmentConfig, error) { return e.config, e.Error() }

type TargetUrlRequirement struct {
	url    Url
	unique bool
}

func NewTargetUrlRequirement(url Url, unique bool) TargetUrlRequirement {
	return TargetUrlRequirement{
		url:    url,
		unique: unique,
	}
}

func (e TargetUrlRequirement) Error() error {
	if !e.unique {
		return ErrUrlAlreadyTaken
	}

	return nil
}

func (e TargetUrlRequirement) Met() (Url, error) { return e.url, e.Error() }

type ProviderConfigRequirement struct {
	config ProviderConfig
	unique bool
}

func NewProviderConfigRequirement(config ProviderConfig, unique bool) ProviderConfigRequirement {
	return ProviderConfigRequirement{
		config: config,
		unique: unique,
	}
}

func (e ProviderConfigRequirement) Error() error {
	if !e.unique {
		return ErrConfigAlreadyTaken
	}

	return nil
}

func (e ProviderConfigRequirement) Met() (ProviderConfig, error) { return e.config, e.Error() }
