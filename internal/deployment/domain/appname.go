package domain

import (
	"regexp"

	"github.com/YuukanOO/seelf/pkg/apperr"
)

var (
	ErrInvalidAppName = apperr.New("invalid_app_name")
	allowedAppnameRe  = regexp.MustCompile("^[a-z0-9-_]+$") // Since the name will be used as a subdomain, we better be restrictive for now
)

type AppName string

// Creates an AppName from a given raw value and returns any error if the value
// is not a valid AppName.
func AppNameFrom(value string) (AppName, error) {
	if !allowedAppnameRe.MatchString(value) {
		return "", ErrInvalidAppName
	}

	return AppName(value), nil
}
