package domain

import (
	"database/sql/driver"
	"encoding/json"
	"net/url"

	"github.com/YuukanOO/seelf/pkg/apperr"
	"github.com/YuukanOO/seelf/pkg/monad"
)

const schemeHttps = "https"

// Url struct which embed an url.URL struct and provides additional methods and meaning.
type Url struct {
	value *url.URL
}

var ErrInvalidUrl = apperr.New("invalid_url")

// Try to parse a raw url into an Url struct.
func UrlFrom(raw string) (Url, error) {
	u, err := url.Parse(raw)

	if err != nil || u.Scheme == "" {
		return Url{}, ErrInvalidUrl
	}

	return Url{u}, nil
}

func (u Url) Host() string          { return u.value.Host }
func (u Url) UseSSL() bool          { return u.value.Scheme == schemeHttps }
func (u Url) Equals(other Url) bool { return u.value.String() == other.value.String() }

// Returns the user part of the url if any.
func (u Url) User() (m monad.Maybe[string]) {
	if u.value.User != nil {
		m.Set(u.value.User.Username())
	}

	return m
}

// Returns the root part of an url.
func (u Url) Root() Url {
	url := *u.value
	url.RawQuery = ""
	url.Path = ""
	return Url{&url}
}

// Returns a new url representing a subdomain.
func (u Url) SubDomain(subdomain string) Url {
	url := *u.value
	// FIXME: should we validate the given subdomain here? Or at least encode it
	url.Host = subdomain + "." + u.Host()
	return Url{&url}
}

// Returns a new url without the user part.
func (u Url) WithoutUser() Url {
	url := *u.value
	url.User = nil
	return Url{&url}
}

func (u Url) String() string { return u.value.String() }

func (u Url) Value() (driver.Value, error) {
	return u.value.String(), nil
}

func (u *Url) Scan(value any) error {
	url, err := UrlFrom(value.(string))

	if err != nil {
		return err
	}

	u.value = url.value

	return nil
}

func (u Url) MarshalJSON() ([]byte, error) {
	return json.Marshal(u.String())
}

func (u *Url) UnmarshalJSON(b []byte) error {
	var str string

	if err := json.Unmarshal(b, &str); err != nil {
		return err
	}

	return u.Scan(str)
}
