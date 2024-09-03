package domain_test

import (
	"testing"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/assert"
	"github.com/YuukanOO/seelf/pkg/must"
)

func Test_Url(t *testing.T) {
	t.Run("should parse the given url on construction", func(t *testing.T) {
		tests := []struct {
			value string
			valid bool
		}{
			{"", false},
			{"something.com", false},
			{"http://something.com", true},
			{"https://something.secure.com", true},
			{"http://127.0.0.1", true},
		}

		for _, test := range tests {
			t.Run(test.value, func(t *testing.T) {
				u, err := domain.UrlFrom(test.value)

				if test.valid {
					assert.Nil(t, err)
					assert.Equal(t, test.value, u.String())
				} else {
					assert.ErrorIs(t, domain.ErrInvalidUrl, err)
				}
			})
		}
	})

	t.Run("should get wether its a secure url or not", func(t *testing.T) {
		httpUrl := must.Panic(domain.UrlFrom("http://something.com"))
		httpsUrl := must.Panic(domain.UrlFrom("https://something.com"))

		assert.False(t, httpUrl.UseSSL())
		assert.True(t, httpsUrl.UseSSL())
	})

	t.Run("should be able to prepend a subdomain", func(t *testing.T) {
		url := must.Panic(domain.UrlFrom("http://something.com"))
		subdomained := url.SubDomain("an-app")

		assert.Equal(t, "http://something.com", url.String())
		assert.Equal(t, "http://an-app.something.com", subdomained.String())
	})

	t.Run("should implement the valuer interface", func(t *testing.T) {
		url := must.Panic(domain.UrlFrom("http://something.com"))
		value, err := url.Value()

		assert.Nil(t, err)
		assert.Equal(t, "http://something.com", value.(string))
	})

	t.Run("should implement the scanner interface", func(t *testing.T) {
		var (
			value = "http://something.com"
			url   domain.Url
		)
		err := url.Scan(value)

		assert.Nil(t, err)
		assert.Equal(t, "http://something.com", url.String())
	})

	t.Run("should marshal to json", func(t *testing.T) {
		url := must.Panic(domain.UrlFrom("http://something.com"))
		json, err := url.MarshalJSON()

		assert.Nil(t, err)
		assert.Equal(t, `"http://something.com"`, string(json))
	})

	t.Run("should unmarshal from json", func(t *testing.T) {
		var (
			value = `"http://something.com"`
			url   domain.Url
		)
		err := url.UnmarshalJSON([]byte(value))

		assert.Nil(t, err)
		assert.Equal(t, "http://something.com", url.String())
	})

	t.Run("should retrieve the user part of an url if any", func(t *testing.T) {
		url := must.Panic(domain.UrlFrom("http://seelf@docker.localhost"))

		assert.True(t, url.User().HasValue())
		assert.Equal(t, "seelf", url.User().MustGet())

		url = must.Panic(domain.UrlFrom("http://docker.localhost"))

		assert.False(t, url.User().HasValue())
	})

	t.Run("should be able to remove the user part of an url", func(t *testing.T) {
		url := must.Panic(domain.UrlFrom("http://seelf@docker.localhost"))

		assert.Equal(t, "http://docker.localhost", url.WithoutUser().String())
		assert.Equal(t, "http://seelf@docker.localhost", url.String())

		url = must.Panic(domain.UrlFrom("http://docker.localhost"))

		assert.Equal(t, "http://docker.localhost", url.WithoutUser().String())
	})

	t.Run("should be able to remove path and query from an url", func(t *testing.T) {
		url := must.Panic(domain.UrlFrom("http://docker.localhost/some/path?query=value"))

		assert.Equal(t, "http://docker.localhost", url.Root().String())
		assert.Equal(t, "http://docker.localhost/some/path?query=value", url.String())
	})
}
