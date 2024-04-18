package domain_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/must"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

func Test_Services(t *testing.T) {
	app := must.Panic(domain.NewApp("my-app",
		domain.NewEnvironmentConfigRequirement(domain.NewEnvironmentConfig("production-target"), true, true),
		domain.NewEnvironmentConfigRequirement(domain.NewEnvironmentConfig("staging-target"), true, true),
		"uid"))
	appidLower := strings.ToLower(string(app.ID()))
	conf := must.Panic(app.ConfigSnapshotFor(domain.Production))

	t.Run("should be able to add a private service", func(t *testing.T) {
		var (
			s     domain.Services
			added domain.Service
		)

		s, added = s.Append(conf, "db", "postgres:14-alpine", false)

		testutil.HasLength(t, s, 1)
		testutil.Equals(t, "db", added.Name())
		testutil.Equals(t, fmt.Sprintf("my-app-production-%s-db", appidLower), added.QualifiedName())
		testutil.Equals(t, "postgres:14-alpine", added.Image())
		testutil.IsFalse(t, added.Subdomain().HasValue())
	})

	t.Run("should be able to add a public service", func(t *testing.T) {
		var (
			s     domain.Services
			added domain.Service
		)

		s, added = s.Append(conf, "app", "my-app-production/app:1", true)

		testutil.HasLength(t, s, 1)
		testutil.Equals(t, "app", added.Name())
		testutil.Equals(t, fmt.Sprintf("my-app-production-%s-app", appidLower), added.QualifiedName())
		testutil.Equals(t, "my-app-production/app:1", added.Image())
		testutil.Equals(t, "my-app", added.Subdomain().MustGet())
	})

	t.Run("should generates an image name if no one is provided", func(t *testing.T) {
		var (
			s     domain.Services
			added domain.Service
		)

		s, added = s.Append(conf, "app", "", true)

		testutil.HasLength(t, s, 1)
		testutil.Equals(t, "app", added.Name())
		testutil.Equals(t, fmt.Sprintf("my-app-production-%s-app", appidLower), added.QualifiedName())
		testutil.Equals(t, fmt.Sprintf("my-app-%s/app:production", appidLower), added.Image())
		testutil.Equals(t, "my-app", added.Subdomain().MustGet())
	})

	t.Run("should handle multiple service on the same domain by adding a prefix if needed", func(t *testing.T) {
		var (
			s       domain.Services
			mainApp domain.Service
			other   domain.Service
		)

		s, _ = s.Append(conf, "db", "postgres:14-alpine", false)
		s, mainApp = s.Append(conf, "app", "my-app-production/app:1", true)
		s, other = s.Append(conf, "other-service", "my-app-production/other-service:1", true)

		testutil.HasLength(t, s, 3)
		testutil.Equals(t, "app", mainApp.Name())
		testutil.Equals(t, "my-app-production/app:1", mainApp.Image())
		testutil.Equals(t, "my-app", mainApp.Subdomain().MustGet())

		testutil.Equals(t, "other-service", other.Name())
		testutil.Equals(t, "my-app-production/other-service:1", other.Image())
		testutil.Equals(t, "other-service.my-app", other.Subdomain().MustGet())
	})

	t.Run("should implement the valuer interface", func(t *testing.T) {
		var services domain.Services

		services, _ = services.Append(conf, "internal", "an/image", false)
		services, _ = services.Append(conf, "public", "another/image", true)

		value, err := services.Value()

		testutil.IsNil(t, err)
		testutil.Equals(t, fmt.Sprintf(`[{"name":"internal","qualified_name":"my-app-production-%s-internal","image":"an/image","subdomain":null},{"name":"public","qualified_name":"my-app-production-%s-public","image":"another/image","subdomain":"my-app"}]`, appidLower, appidLower), value.(string))
	})

	t.Run("should implement the scanner interface", func(t *testing.T) {
		var services domain.Services

		err := services.Scan(`[{"name":"internal","qualified_name":"my-app-production-internal","image":"an/image","subdomain":null},{"name":"public","qualified_name":"my-app-production-public","image":"another/image","subdomain":"my-app"}]`)

		testutil.IsNil(t, err)
		testutil.HasLength(t, services, 2)

		s := services[0]
		testutil.IsFalse(t, s.Subdomain().HasValue())
		testutil.Equals(t, "internal", s.Name())
		testutil.Equals(t, "my-app-production-internal", s.QualifiedName())
		testutil.Equals(t, "an/image", s.Image())

		s = services[1]
		testutil.IsTrue(t, s.Subdomain().HasValue())
		testutil.Equals(t, "public", s.Name())
		testutil.Equals(t, "my-app-production-public", s.QualifiedName())
		testutil.Equals(t, "another/image", s.Image())
		testutil.Equals(t, "my-app", s.Subdomain().MustGet())
	})
}
