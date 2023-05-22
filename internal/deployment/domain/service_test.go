package domain_test

import (
	"testing"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

func Test_Services(t *testing.T) {
	app := domain.NewApp("my-app", "uid")
	domainUrl, _ := domain.UrlFrom("http://docker.localhost")
	conf := domain.NewConfig(app, "production")

	t.Run("should be able to add a private service", func(t *testing.T) {
		var (
			s     domain.Services
			added domain.Service
		)

		s, added = s.Internal(conf, "db", "postgres:14-alpine")

		testutil.HasLength(t, s, 1)
		testutil.Equals(t, "db", added.Name())
		testutil.Equals(t, "my-app-production-db", added.QualifiedName())
		testutil.Equals(t, "postgres:14-alpine", added.Image())
		testutil.IsFalse(t, added.Url().HasValue())
	})

	t.Run("should be able to add a public service", func(t *testing.T) {
		var (
			s     domain.Services
			added domain.Service
		)

		s, added = s.Public(domainUrl, conf, "app", "my-app-production/app:1")

		testutil.HasLength(t, s, 1)
		testutil.Equals(t, "app", added.Name())
		testutil.Equals(t, "my-app-production-app", added.QualifiedName())
		testutil.Equals(t, "my-app-production/app:1", added.Image())
		testutil.Equals(t, "http://my-app.docker.localhost", added.Url().MustGet().String())
	})

	t.Run("should generates an image name if no one is provided", func(t *testing.T) {
		var (
			s     domain.Services
			added domain.Service
		)

		s, added = s.Public(domainUrl, conf, "app", "")

		testutil.HasLength(t, s, 1)
		testutil.Equals(t, "app", added.Name())
		testutil.Equals(t, "my-app-production-app", added.QualifiedName())
		testutil.Equals(t, "my-app/app:production", added.Image())
		testutil.Equals(t, "http://my-app.docker.localhost", added.Url().MustGet().String())
	})

	t.Run("should handle multiple service on the same domain by adding a prefix if needed", func(t *testing.T) {
		var (
			s       domain.Services
			mainApp domain.Service
			other   domain.Service
		)

		s, _ = s.Internal(conf, "db", "postgres:14-alpine")
		s, mainApp = s.Public(domainUrl, conf, "app", "my-app-production/app:1")
		s, other = s.Public(domainUrl, conf, "other-service", "my-app-production/other-service:1")

		testutil.HasLength(t, s, 3)
		testutil.Equals(t, "app", mainApp.Name())
		testutil.Equals(t, "my-app-production/app:1", mainApp.Image())
		testutil.Equals(t, "http://my-app.docker.localhost", mainApp.Url().MustGet().String())

		testutil.Equals(t, "other-service", other.Name())
		testutil.Equals(t, "my-app-production/other-service:1", other.Image())
		testutil.Equals(t, "http://other-service.my-app.docker.localhost", other.Url().MustGet().String())
	})

	t.Run("should implement the valuer interface", func(t *testing.T) {
		var services domain.Services

		services, _ = services.Internal(conf, "internal", "an/image")
		services, _ = services.Public(domainUrl, conf, "public", "another/image")

		value, err := services.Value()

		testutil.IsNil(t, err)
		testutil.Equals(t, `[{"name":"internal","qualified_name":"my-app-production-internal","image":"an/image","url":null},{"name":"public","qualified_name":"my-app-production-public","image":"another/image","url":"http://my-app.docker.localhost"}]`, value.(string))
	})

	t.Run("should implement the scanner interface", func(t *testing.T) {
		var services domain.Services

		err := services.Scan(`[{"name":"internal","qualified_name":"my-app-production-internal","image":"an/image","url":null},{"name":"public","qualified_name":"my-app-production-public","image":"another/image","url":"http://my-app.docker.localhost"}]`)

		testutil.IsNil(t, err)
		testutil.HasLength(t, services, 2)

		s := services[0]
		testutil.IsFalse(t, s.Url().HasValue())
		testutil.Equals(t, "internal", s.Name())
		testutil.Equals(t, "my-app-production-internal", s.QualifiedName())
		testutil.Equals(t, "an/image", s.Image())

		s = services[1]
		testutil.IsTrue(t, s.Url().HasValue())
		testutil.Equals(t, "public", s.Name())
		testutil.Equals(t, "my-app-production-public", s.QualifiedName())
		testutil.Equals(t, "another/image", s.Image())
		url := s.Url().MustGet()
		testutil.Equals(t, "http://my-app.docker.localhost", url.String())
	})
}
