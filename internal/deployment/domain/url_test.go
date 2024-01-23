package domain_test

import (
	"testing"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/must"
	"github.com/YuukanOO/seelf/pkg/testutil"
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
		}

		for _, test := range tests {
			t.Run(test.value, func(t *testing.T) {
				u, err := domain.UrlFrom(test.value)

				if test.valid {
					testutil.IsNil(t, err)
					testutil.Equals(t, test.value, u.String())
				} else {
					testutil.ErrorIs(t, domain.ErrInvalidUrl, err)
				}
			})
		}
	})

	t.Run("should get wether its a secure url or not", func(t *testing.T) {
		httpUrl, _ := domain.UrlFrom("http://something.com")
		httpsUrl, _ := domain.UrlFrom("https://something.com")

		testutil.IsFalse(t, httpUrl.UseSSL())
		testutil.IsTrue(t, httpsUrl.UseSSL())
	})

	t.Run("should be able to prepend a subdomain", func(t *testing.T) {
		url, _ := domain.UrlFrom("http://something.com")
		subdomained := url.SubDomain("an-app")

		testutil.Equals(t, "http://something.com", url.String())
		testutil.Equals(t, "http://an-app.something.com", subdomained.String())
	})

	t.Run("should implement the valuer interface", func(t *testing.T) {
		url, _ := domain.UrlFrom("http://something.com")
		value, err := url.Value()

		testutil.IsNil(t, err)
		testutil.Equals(t, "http://something.com", value.(string))
	})

	t.Run("should implement the scanner interface", func(t *testing.T) {
		var (
			value = "http://something.com"
			url   domain.Url
		)
		err := url.Scan(value)

		testutil.IsNil(t, err)
		testutil.Equals(t, "http://something.com", url.String())
	})

	t.Run("should marshal to json", func(t *testing.T) {
		url, _ := domain.UrlFrom("http://something.com")
		json, err := url.MarshalJSON()

		testutil.IsNil(t, err)
		testutil.Equals(t, `"http://something.com"`, string(json))
	})

	t.Run("should unmarshal from json", func(t *testing.T) {
		var (
			value = `"http://something.com"`
			url   domain.Url
		)
		err := url.UnmarshalJSON([]byte(value))

		testutil.IsNil(t, err)
		testutil.Equals(t, "http://something.com", url.String())
	})

	t.Run("should retrieve the user part of an url if any", func(t *testing.T) {
		url := must.Panic(domain.UrlFrom("http://seelf@docker.localhost"))

		testutil.IsTrue(t, url.User().HasValue())
		testutil.Equals(t, "seelf", url.User().MustGet())

		url = must.Panic(domain.UrlFrom("http://docker.localhost"))

		testutil.IsFalse(t, url.User().HasValue())
	})

	t.Run("should be able to remove the user part of an url", func(t *testing.T) {
		url := must.Panic(domain.UrlFrom("http://seelf@docker.localhost"))

		testutil.Equals(t, "http://docker.localhost", url.WithoutUser().String())

		url = must.Panic(domain.UrlFrom("http://docker.localhost"))

		testutil.Equals(t, "http://docker.localhost", url.WithoutUser().String())
	})

	t.Run("should be able to remove path and query from an url", func(t *testing.T) {
		url := must.Panic(domain.UrlFrom("http://docker.localhost/some/path?query=value"))

		testutil.Equals(t, "http://docker.localhost", url.Root().String())
	})
}
