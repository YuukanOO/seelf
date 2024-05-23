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
	value url.URL
	user  monad.Maybe[url.Userinfo]
}

var ErrInvalidUrl = apperr.New("invalid_url")

// Try to parse a raw url into an Url struct.
func UrlFrom(raw string) (Url, error) {
	u, err := url.Parse(raw)

	if err != nil || u.Scheme == "" {
		return Url{}, ErrInvalidUrl
	}

	var result Url

	// We want to get rid of the pointer part so equality could work without specific handling
	if u.User != nil {
		result.user.Set(*u.User)
		u.User = nil
	}

	result.value = *u

	return result, nil
}

func (u Url) Host() string { return u.value.Host }
func (u Url) UseSSL() bool { return u.value.Scheme == schemeHttps }

// Returns the user part of the url if any.
func (u Url) User() (m monad.Maybe[string]) {
	if usr, hasUser := u.user.TryGet(); hasUser {
		m.Set(usr.Username())
	}

	return m
}

// Returns the root part of an url.
func (u Url) Root() Url {
	u.value.RawQuery = ""
	u.value.Path = ""
	return u
}

// Returns a new url representing a subdomain.
func (u Url) SubDomain(subdomain string) Url {
	// FIXME: should we validate the given subdomain here? Or at least encode it
	u.value.Host = subdomain + "." + u.Host()
	return u
}

// Returns a new url without the user part.
func (u Url) WithoutUser() Url {
	u.user.Unset()
	return u
}

func (u Url) String() string {
	raw := u.value

	if usr, hasUser := u.user.TryGet(); hasUser {
		raw.User = &usr
	}

	return raw.String()
}

func (u Url) Value() (driver.Value, error) {
	return u.value.String(), nil
}

func (u *Url) Scan(value any) error {
	url, err := UrlFrom(value.(string))

	if err != nil {
		return err
	}

	*u = url

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
