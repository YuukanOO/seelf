package domain_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/deployment/fixture"
	"github.com/YuukanOO/seelf/pkg/assert"
)

func Test_ServicesBuilder(t *testing.T) {

	t.Run("could be created from a deployment configuration", func(t *testing.T) {
		t.Run("should use the given image if any", func(t *testing.T) {
			app := fixture.App(fixture.WithAppName("my-app"))
			deployment := fixture.Deployment(fixture.FromApp(app))

			builder := deployment.Config().ServicesBuilder()
			service := builder.AddService("db", "postgres:14-alpine")

			assert.Equal(t, "db", service.Name())
			assert.Equal(t, "postgres:14-alpine", service.Image())
		})

		t.Run("should generate a unique image name if not set", func(t *testing.T) {
			app := fixture.App(fixture.WithAppName("my-app"))
			appidLower := strings.ToLower(string(app.ID()))
			deployment := fixture.Deployment(fixture.FromApp(app))
			builder := deployment.Config().ServicesBuilder()

			service := builder.AddService("app", "")

			assert.Equal(t, "app", service.Name())
			assert.Equal(t, fmt.Sprintf("my-app-%s/app:production", appidLower), service.Image())
		})
	})

	t.Run("should returns an existing service if trying to add one with the same name", func(t *testing.T) {
		app := fixture.App(fixture.WithAppName("my-app"))
		deployment := fixture.Deployment(fixture.FromApp(app))

		builder := deployment.Config().ServicesBuilder()

		one := builder.AddService("app", "")
		two := builder.AddService("app", "")

		assert.HasLength(t, 1, builder.Services())
		assert.Equal(t, one, two)
	})

	t.Run("should returns the existing entrypoint if trying to add one for the same router and port", func(t *testing.T) {
		app := fixture.App(fixture.WithAppName("my-app"))
		deployment := fixture.Deployment(fixture.FromApp(app))

		builder := deployment.Config().ServicesBuilder()
		service := builder.AddService("app", "image")
		entrypointOne := service.AddHttpEntrypoint(80, true)
		entrypointTwo := service.AddHttpEntrypoint(80, false)

		assert.HasLength(t, 1, builder.Services())
		assert.Equal(t, 1, len(builder.Services().Entrypoints()))
		assert.True(t, builder.Services().Entrypoints()[0].IsCustom())
		assert.Equal(t, entrypointOne, entrypointTwo)
	})

	t.Run("could have http entrypoints added", func(t *testing.T) {
		app := fixture.App(fixture.WithAppName("my-app"))
		appidLower := strings.ToLower(string(app.ID()))
		deployment := fixture.Deployment(fixture.FromApp(app))

		builder := deployment.Config().ServicesBuilder()
		service := builder.AddService("app", "")
		service.AddHttpEntrypoint(80, true)
		service.AddHttpEntrypoint(8080, false)
		service = builder.AddService("other", "")
		service.AddHttpEntrypoint(3000, false)

		services := builder.Services()
		assert.HasLength(t, 2, services)
		assert.Equal(t, "app", services[0].Name())
		assert.Equal(t, fmt.Sprintf("my-app-%s/app:production", appidLower), services[0].Image())

		assert.Equal(t, "other", services[1].Name())
		assert.Equal(t, fmt.Sprintf("my-app-%s/other:production", appidLower), services[1].Image())

		entrypoints := services.Entrypoints()
		assert.HasLength(t, 3, entrypoints)

		assert.Equal(t, fmt.Sprintf("my-app-production-%s-app-80-http", appidLower), string(entrypoints[0].Name()))
		assert.Equal(t, domain.RouterHttp, entrypoints[0].Router())
		assert.True(t, entrypoints[0].IsCustom())
		assert.Equal(t, 80, entrypoints[0].Port())
		assert.Equal(t, "my-app", entrypoints[0].Subdomain().Get(""))

		assert.Equal(t, fmt.Sprintf("my-app-production-%s-app-8080-http", appidLower), string(entrypoints[1].Name()))
		assert.Equal(t, domain.RouterHttp, entrypoints[1].Router())
		assert.False(t, entrypoints[1].IsCustom())
		assert.Equal(t, 8080, entrypoints[1].Port())
		assert.Equal(t, "my-app", entrypoints[1].Subdomain().Get(""))

		assert.Equal(t, fmt.Sprintf("my-app-production-%s-other-3000-http", appidLower), string(entrypoints[2].Name()))
		assert.Equal(t, domain.RouterHttp, entrypoints[2].Router())
		assert.False(t, entrypoints[2].IsCustom())
		assert.Equal(t, 3000, entrypoints[2].Port())
		assert.Equal(t, "other.my-app", entrypoints[2].Subdomain().Get(""))
	})

	t.Run("could have one or more TCP/UDP entrypoints attached", func(t *testing.T) {
		app := fixture.App(fixture.WithAppName("my-app"))
		appidLower := strings.ToLower(string(app.ID()))
		deployment := fixture.Deployment(fixture.FromApp(app))

		builder := deployment.Config().ServicesBuilder()

		service := builder.AddService("app", "")

		tcp := service.AddTCPEntrypoint(8080, true)
		assert.Equal(t, fmt.Sprintf("my-app-production-%s-app-8080-tcp", appidLower), string(tcp.Name()))
		assert.Equal(t, domain.RouterTcp, tcp.Router())
		assert.True(t, tcp.IsCustom())
		assert.False(t, tcp.Subdomain().HasValue())
		assert.Equal(t, 8080, tcp.Port())

		udp := service.AddUDPEntrypoint(8080, true)
		assert.Equal(t, fmt.Sprintf("my-app-production-%s-app-8080-udp", appidLower), string(udp.Name()))
		assert.Equal(t, domain.RouterUdp, udp.Router())
		assert.True(t, udp.IsCustom())
		assert.False(t, udp.Subdomain().HasValue())
		assert.Equal(t, 8080, udp.Port())
	})
}

func Test_Services(t *testing.T) {

	t.Run("should be able to return all entrypoints", func(t *testing.T) {
		deployment := fixture.Deployment()
		builder := deployment.Config().ServicesBuilder()

		service := builder.AddService("app", "")
		http := service.AddHttpEntrypoint(80, false)
		udp := service.AddUDPEntrypoint(8080, true)
		service = builder.AddService("db", "postgres:14-alpine")
		tcp := service.AddTCPEntrypoint(5432, true)
		builder.AddService("cache", "redis:6-alpine")
		services := builder.Services()

		entrypoints := services.Entrypoints()

		assert.HasLength(t, 3, entrypoints)
		assert.Equal(t, http, entrypoints[0])
		assert.Equal(t, udp, entrypoints[1])
		assert.Equal(t, tcp, entrypoints[2])
	})

	t.Run("should be able to return all custom entrypoints", func(t *testing.T) {
		deployment := fixture.Deployment()
		builder := deployment.Config().ServicesBuilder()

		service := builder.AddService("app", "")
		service.AddHttpEntrypoint(80, false)
		udp := service.AddUDPEntrypoint(8080, true)
		service = builder.AddService("db", "postgres:14-alpine")
		tcp := service.AddTCPEntrypoint(5432, true)
		builder.AddService("cache", "redis:6-alpine")
		services := builder.Services()

		entrypoints := services.CustomEntrypoints()

		assert.HasLength(t, 2, entrypoints)
		assert.Equal(t, udp, entrypoints[0])
		assert.Equal(t, tcp, entrypoints[1])
	})

	t.Run("should implement the valuer interface", func(t *testing.T) {
		app := fixture.App(fixture.WithAppName("my-app"))
		deployment := fixture.Deployment(fixture.FromApp(app))
		appidLower := strings.ToLower(string(deployment.ID().AppID()))
		builder := deployment.Config().ServicesBuilder()

		service := builder.AddService("app", "")
		service.AddHttpEntrypoint(80, false)
		service.AddTCPEntrypoint(8080, true)
		service = builder.AddService("db", "postgres:14-alpine")
		service.AddTCPEntrypoint(5432, true)
		builder.AddService("cache", "redis:6-alpine")

		value, err := builder.Services().Value()

		assert.Nil(t, err)
		assert.Equal(t, fmt.Sprintf(`[{"name":"app","image":"my-app-%s/app:production","entrypoints":[{"name":"my-app-production-%s-app-80-http","is_custom":false,"router":"http","subdomain":"my-app","port":80},{"name":"my-app-production-%s-app-8080-tcp","is_custom":true,"router":"tcp","subdomain":null,"port":8080}]},{"name":"db","image":"postgres:14-alpine","entrypoints":[{"name":"my-app-production-%s-db-5432-tcp","is_custom":true,"router":"tcp","subdomain":null,"port":5432}]},{"name":"cache","image":"redis:6-alpine","entrypoints":[]}]`,
			appidLower, appidLower, appidLower, appidLower), value.(string))
	})

	t.Run("should implement the scanner interface", func(t *testing.T) {
		var services domain.Services

		err := services.Scan(`[
  {
    "name": "app",
    "qualified_name": "my-app-production-2fa8domd2sh7ehyqlxf7jvj57xs-app",
    "image": "my-app-2fa8domd2sh7ehyqlxf7jvj57xs/app:production",
    "entrypoints": [
      {
        "name": "my-app-production-2fa8domd2sh7ehyqlxf7jvj57xs-app-80-http",
		"is_custom": false,
        "router": "http",
        "subdomain": "my-app",
        "port": 80
      },
      {
        "name": "my-app-production-2fa8domd2sh7ehyqlxf7jvj57xs-app-8080-tcp",
		"is_custom": true,
        "router": "tcp",
        "subdomain": null,
        "port": 8080
      }
    ]
  },
  {
    "name": "db",
    "qualified_name": "my-app-production-2fa8domd2sh7ehyqlxf7jvj57xs-db",
    "image": "postgres:14-alpine",
    "entrypoints": [
      {
        "name": "my-app-production-2fa8domd2sh7ehyqlxf7jvj57xs-db-5432-tcp",
		"is_custom": true,
        "router": "tcp",
        "subdomain": null,
        "port": 5432
      }
    ]
  },
  {
    "name": "cache",
    "qualified_name": "my-app-production-2fa8domd2sh7ehyqlxf7jvj57xs-cache",
    "image": "redis:6-alpine",
    "entrypoints": []
  }
]`)

		assert.Nil(t, err)
		assert.HasLength(t, 3, services)

		v, err := services.Value()

		assert.Nil(t, err)
		assert.Equal(t, `[{"name":"app","image":"my-app-2fa8domd2sh7ehyqlxf7jvj57xs/app:production","entrypoints":[{"name":"my-app-production-2fa8domd2sh7ehyqlxf7jvj57xs-app-80-http","is_custom":false,"router":"http","subdomain":"my-app","port":80},{"name":"my-app-production-2fa8domd2sh7ehyqlxf7jvj57xs-app-8080-tcp","is_custom":true,"router":"tcp","subdomain":null,"port":8080}]},{"name":"db","image":"postgres:14-alpine","entrypoints":[{"name":"my-app-production-2fa8domd2sh7ehyqlxf7jvj57xs-db-5432-tcp","is_custom":true,"router":"tcp","subdomain":null,"port":5432}]},{"name":"cache","image":"redis:6-alpine","entrypoints":[]}]`, v.(string))
	})
}

func Test_Port(t *testing.T) {
	t.Run("should be able to parse a port from a raw string value", func(t *testing.T) {
		_, err := domain.ParsePort("failed")

		assert.ErrorIs(t, domain.ErrInvalidPort, err)

		p, err := domain.ParsePort("8080")
		assert.Nil(t, err)
		assert.Equal(t, 8080, p)
	})

	t.Run("should convert the port to a string", func(t *testing.T) {
		p := domain.Port(8080)
		assert.Equal(t, "8080", p.String())
	})

	t.Run("should convert the port to a uint32", func(t *testing.T) {
		p := domain.Port(8080)
		assert.Equal(t, 8080, p.Uint32())
	})
}

func Test_EntrypointName(t *testing.T) {
	t.Run("should provide a protocol", func(t *testing.T) {
		assert.Equal(t, "tcp", domain.EntrypointName("my-app-production-2fa8domd2sh7ehyqlxf7jvj57xs-app-8080-http").Protocol())
		assert.Equal(t, "tcp", domain.EntrypointName("my-app-production-2fa8domd2sh7ehyqlxf7jvj57xs-app-8080-tcp").Protocol())
		assert.Equal(t, "udp", domain.EntrypointName("my-app-production-2fa8domd2sh7ehyqlxf7jvj57xs-app-8080-udp").Protocol())
	})
}
