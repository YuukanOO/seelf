package domain

import (
	"github.com/YuukanOO/seelf/pkg/monad"
)

// Holds the vcs configuration of an application.
type VCSConfig struct {
	url   Url
	token monad.Maybe[string]
}

// Instantiates a new version control config object.
func NewVCSConfig(url Url) VCSConfig {
	return VCSConfig{
		url: url,
	}
}

// If this repository needs authentication, use the provided token.
func (c VCSConfig) Authenticated(token string) VCSConfig {
	c.token = c.token.WithValue(token)
	return c
}

// Returns a new VCS Config without the token.
func (c VCSConfig) Public() VCSConfig {
	c.token = c.token.None()
	return c
}

// Updates the VCS Config with the provided url.
func (c VCSConfig) WithUrl(url Url) VCSConfig {
	c.url = url
	return c
}

func (c VCSConfig) Url() Url                   { return c.url }
func (c VCSConfig) Token() monad.Maybe[string] { return c.token }

func (c VCSConfig) Equals(other VCSConfig) bool {
	return other.url.String() == c.url.String() && other.token == c.token
}
