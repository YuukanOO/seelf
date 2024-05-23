package domain_test

import (
	"fmt"
	"testing"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

func Test_VersionControl(t *testing.T) {
	t.Run("should be created from a valid url", func(t *testing.T) {
		url, _ := domain.UrlFrom("http://somewhere.git")

		conf := domain.NewVersionControl(url)

		testutil.Equals(t, url, conf.Url())
		testutil.IsFalse(t, conf.Token().HasValue())
	})

	t.Run("should hold a token if authentication is needed", func(t *testing.T) {
		var (
			url, _        = domain.UrlFrom("http://somewhere.git")
			token  string = "some token"
		)

		conf := domain.NewVersionControl(url)
		conf.Authenticated(token)

		testutil.Equals(t, url, conf.Url())
		testutil.Equals(t, token, conf.Token().Get(""))
	})

	t.Run("could update the url", func(t *testing.T) {
		var (
			url, _           = domain.UrlFrom("http://somewhere.git")
			newUrl, _        = domain.UrlFrom("http://somewhere.else.git")
			token     string = "some token"
		)

		conf := domain.NewVersionControl(url)
		conf.Authenticated(token)
		conf.HasUrl(newUrl)

		testutil.Equals(t, newUrl, conf.Url())
		testutil.Equals(t, token, conf.Token().Get(""))
	})

	t.Run("could remove a token", func(t *testing.T) {
		url, _ := domain.UrlFrom("http://somewhere.git")

		conf := domain.NewVersionControl(url)
		conf.Authenticated("a token")
		conf.Public()

		testutil.Equals(t, url, conf.Url())
		testutil.IsFalse(t, conf.Token().HasValue())
	})

	t.Run("should be able to compare itself with another config", func(t *testing.T) {
		var (
			url, _                           = domain.UrlFrom("http://somewhere.git")
			sameUrlDifferentStruct, _        = domain.UrlFrom("http://somewhere.git")
			anotherUrl, _                    = domain.UrlFrom("http://somewhere-else.git")
			token                     string = "some token"
			anotherToken              string = "another token"
		)

		tests := []struct {
			first    func() domain.VersionControl
			second   func() domain.VersionControl
			expected bool
		}{
			{
				func() domain.VersionControl {
					conf := domain.NewVersionControl(url)
					conf.Authenticated(token)
					return conf
				},
				func() domain.VersionControl {
					return domain.NewVersionControl(sameUrlDifferentStruct)
				},
				false,
			},
			{
				func() domain.VersionControl {
					return domain.NewVersionControl(url)
				},
				func() domain.VersionControl {
					return domain.NewVersionControl(anotherUrl)
				},
				false,
			},
			{
				func() domain.VersionControl {
					conf := domain.NewVersionControl(url)
					conf.Authenticated(token)
					return conf
				},
				func() domain.VersionControl {
					conf := domain.NewVersionControl(sameUrlDifferentStruct)
					conf.Authenticated(anotherToken)
					return conf
				},
				false,
			},
			{
				func() domain.VersionControl {
					return domain.NewVersionControl(url)
				},
				func() domain.VersionControl {
					return domain.NewVersionControl(sameUrlDifferentStruct)
				},
				true,
			},
			{
				func() domain.VersionControl {
					conf := domain.NewVersionControl(url)
					conf.Authenticated(token)
					return conf
				},
				func() domain.VersionControl {
					conf := domain.NewVersionControl(sameUrlDifferentStruct)
					conf.Authenticated(token)
					return conf
				},
				true,
			},
		}

		for _, tt := range tests {
			f := tt.first()
			s := tt.second()
			t.Run(fmt.Sprintf("%v %v", f, s), func(t *testing.T) {
				testutil.Equals(t, tt.expected, f == s)
			})
		}
	})
}
