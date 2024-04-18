package domain

import (
	"github.com/YuukanOO/seelf/pkg/monad"
)

// Holds the vcs configuration of an application.
type VersionControl struct {
	url   Url
	token monad.Maybe[string]
}

// Instantiates a new version control config object.
func NewVersionControl(url Url) VersionControl {
	return VersionControl{
		url: url,
	}
}

// If this repository needs authentication, use the provided token.
func (c *VersionControl) Authenticated(token string) {
	c.token.Set(token)
}

// Mark this repository as public (no authentication needed).
func (c *VersionControl) Public() {
	c.token.Unset()
}

// Updates the vcs url.
func (c *VersionControl) HasUrl(url Url) {
	c.url = url
}

func (c VersionControl) Url() Url                   { return c.url }
func (c VersionControl) Token() monad.Maybe[string] { return c.token }

func (c VersionControl) Equals(other VersionControl) bool {
	return other.url.Equals(c.url) && other.token == c.token
}
