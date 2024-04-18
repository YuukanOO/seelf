package domain

import (
	"regexp"
	"strings"

	"github.com/YuukanOO/seelf/pkg/apperr"
)

var (
	ErrInvalidAppName   = apperr.New("invalid_app_name")
	allowedAppnameChars = regexp.MustCompile("^[a-z0-9-_]+$") // Since the name will be used as a subdomain, we better be restrictive for now
)

const stagingSuffix = "-" + string(Staging) // Staging suffix used to prevent appname to ends with this to avoid domain collisions

type AppName string

// Creates an AppName from a given raw value and returns any error if the value
// is not a valid AppName.
func AppNameFrom(value string) (AppName, error) {
	if !allowedAppnameChars.MatchString(value) || strings.HasSuffix(value, stagingSuffix) {
		return "", ErrInvalidAppName
	}

	return AppName(value), nil
}
