package domain_test

import (
	"testing"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

func Test_VCSConfig(t *testing.T) {
	t.Run("should be created from a valid vcs url", func(t *testing.T) {
		url, _ := domain.UrlFrom("http://somewhere.git")

		conf := domain.NewVCSConfig(url)

		testutil.Equals(t, url, conf.Url())
		testutil.IsFalse(t, conf.Token().HasValue())
	})

	t.Run("should hold a token if authentication is needed", func(t *testing.T) {
		var (
			url, _        = domain.UrlFrom("http://somewhere.git")
			token  string = "some token"
		)

		conf := domain.NewVCSConfig(url).Authenticated(token)

		testutil.Equals(t, url, conf.Url())
		testutil.Equals(t, token, conf.Token().Get(""))
	})

	t.Run("could returns the same config with another url", func(t *testing.T) {
		var (
			url, _           = domain.UrlFrom("http://somewhere.git")
			newUrl, _        = domain.UrlFrom("http://somewhere.else.git")
			token     string = "some token"
		)

		conf := domain.NewVCSConfig(url).Authenticated(token)
		conf = conf.WithUrl(newUrl)

		testutil.Equals(t, newUrl, conf.Url())
		testutil.Equals(t, token, conf.Token().Get(""))
	})

	t.Run("could remove a token", func(t *testing.T) {
		url, _ := domain.UrlFrom("http://somewhere.git")

		conf := domain.NewVCSConfig(url).Authenticated("a token")
		conf = conf.Public()

		testutil.Equals(t, url, conf.Url())
		testutil.IsFalse(t, conf.Token().HasValue())
	})

	t.Run("should be able to compare itself with another vcs config", func(t *testing.T) {
		var (
			url, _                           = domain.UrlFrom("http://somewhere.git")
			sameUrlDifferentStruct, _        = domain.UrlFrom("http://somewhere.git")
			anotherUrl, _                    = domain.UrlFrom("http://somewhere-else.git")
			token                     string = "some token"
			anotherToken              string = "another token"
		)

		testutil.IsFalse(t, domain.NewVCSConfig(url).Authenticated(token).Equals(domain.NewVCSConfig(sameUrlDifferentStruct)))
		testutil.IsFalse(t, domain.NewVCSConfig(url).Equals(domain.NewVCSConfig(anotherUrl)))
		testutil.IsFalse(t, domain.NewVCSConfig(url).Authenticated(token).Equals(domain.NewVCSConfig(sameUrlDifferentStruct).Authenticated(anotherToken)))
		testutil.IsTrue(t, domain.NewVCSConfig(url).Equals(domain.NewVCSConfig(sameUrlDifferentStruct)))
		testutil.IsTrue(t, domain.NewVCSConfig(url).Authenticated(token).Equals(domain.NewVCSConfig(sameUrlDifferentStruct).Authenticated(token)))
	})
}
