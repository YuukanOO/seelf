package domain_test

import (
	"testing"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/assert"
)

func Test_VersionControl(t *testing.T) {
	t.Run("should be created from a valid url", func(t *testing.T) {
		url, _ := domain.UrlFrom("http://somewhere.git")

		conf := domain.NewVersionControl(url)

		assert.Equal(t, url, conf.Url())
		assert.False(t, conf.Token().HasValue())
	})

	t.Run("should hold a token if authentication is needed", func(t *testing.T) {
		var (
			url, _        = domain.UrlFrom("http://somewhere.git")
			token  string = "some token"
		)

		conf := domain.NewVersionControl(url)
		conf.Authenticated(token)

		assert.Equal(t, url, conf.Url())
		assert.Equal(t, token, conf.Token().Get(""))
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

		assert.Equal(t, newUrl, conf.Url())
		assert.Equal(t, token, conf.Token().Get(""))
	})

	t.Run("could remove a token", func(t *testing.T) {
		url, _ := domain.UrlFrom("http://somewhere.git")

		conf := domain.NewVersionControl(url)
		conf.Authenticated("a token")
		conf.Public()

		assert.Equal(t, url, conf.Url())
		assert.False(t, conf.Token().HasValue())
	})
}
