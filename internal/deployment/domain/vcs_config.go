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
func (c *VCSConfig) Authenticated(token string) {
	c.token.Set(token)
}

// Mark this repository as public (no authentication needed).
func (c *VCSConfig) Public() {
	c.token.Unset()
}

// Updates the vcs url.
func (c *VCSConfig) HasUrl(url Url) {
	c.url = url
}

func (c VCSConfig) Url() Url                   { return c.url }
func (c VCSConfig) Token() monad.Maybe[string] { return c.token }

func (c VCSConfig) Equals(other VCSConfig) bool {
	return other.url.String() == c.url.String() && other.token == c.token
}
