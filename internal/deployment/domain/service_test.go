package domain_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/deployment/fixture"
	"github.com/YuukanOO/seelf/pkg/assert"
)

func Test_Service(t *testing.T) {

	t.Run("could be created from a deployment configuration", func(t *testing.T) {
		app := fixture.App(fixture.WithAppName("my-app"))
		appidLower := strings.ToLower(string(app.ID()))
		deployment := fixture.Deployment(fixture.FromApp(app))

		s := deployment.Config().NewService("db", "postgres:14-alpine")

		assert.Equal(t, "db", s.Name())
		assert.Equal(t, "postgres:14-alpine", s.Image())

		s = deployment.Config().NewService("app", "")

		assert.Equal(t, "app", s.Name())
		assert.Equal(t, fmt.Sprintf("my-app-%s/app:production", appidLower), s.Image())
	})

	t.Run("should populate the subdomain when adding HTTP entrypoints", func(t *testing.T) {
		app := fixture.App(fixture.WithAppName("my-app"))
		appidLower := strings.ToLower(string(app.ID()))
		deployment := fixture.Deployment(fixture.FromApp(app))

		s := deployment.Config().NewService("app", "")
		e := s.AddHttpEntrypoint(deployment.Config(), 80, domain.HttpEntrypointOptions{
			Managed:             true,
			UseDefaultSubdomain: true,
		})
		assert.Equal(t, fmt.Sprintf("my-app-production-%s-app-80-http", appidLower), string(e.Name()))
		assert.Equal(t, domain.RouterHttp, e.Router())
		assert.False(t, e.IsCustom())
		assert.Equal(t, "my-app", e.Subdomain().Get(""))
		assert.Equal(t, 80, e.Port())

		e = s.AddHttpEntrypoint(deployment.Config(), 8080, domain.HttpEntrypointOptions{})
		assert.Equal(t, fmt.Sprintf("my-app-production-%s-app-8080-http", appidLower), string(e.Name()))
		assert.Equal(t, domain.RouterHttp, e.Router())
		assert.True(t, e.IsCustom())
		assert.Equal(t, "my-app", e.Subdomain().Get(""))
		assert.Equal(t, 8080, e.Port())

		same := s.AddHttpEntrypoint(deployment.Config(), 8080, domain.HttpEntrypointOptions{})
		assert.Equal(t, e, same)
	})

	t.Run("could have one or more TCP/UDP entrypoints attached", func(t *testing.T) {
		app := fixture.App(fixture.WithAppName("my-app"))
		appidLower := strings.ToLower(string(app.ID()))
		deployment := fixture.Deployment(fixture.FromApp(app))
		s := deployment.Config().NewService("app", "")

		tcp := s.AddTCPEntrypoint(8080)
		assert.Equal(t, fmt.Sprintf("my-app-production-%s-app-8080-tcp", appidLower), string(tcp.Name()))
		assert.Equal(t, domain.RouterTcp, tcp.Router())
		assert.True(t, tcp.IsCustom())
		assert.False(t, tcp.Subdomain().HasValue())
		assert.Equal(t, 8080, tcp.Port())

		udp := s.AddUDPEntrypoint(8080)
		assert.Equal(t, fmt.Sprintf("my-app-production-%s-app-8080-udp", appidLower), string(udp.Name()))
		assert.Equal(t, domain.RouterUdp, udp.Router())
		assert.True(t, udp.IsCustom())
		assert.False(t, udp.Subdomain().HasValue())
		assert.Equal(t, 8080, udp.Port())

		same := s.AddTCPEntrypoint(8080)
		assert.Equal(t, tcp, same)

		same = s.AddUDPEntrypoint(8080)
		assert.Equal(t, udp, same)
	})
}

func Test_Services(t *testing.T) {

	t.Run("should be able to return all entrypoints", func(t *testing.T) {
		deployment := fixture.Deployment()
		var services domain.Services

		s := deployment.Config().NewService("app", "")
		http := s.AddHttpEntrypoint(deployment.Config(), 80, domain.HttpEntrypointOptions{
			Managed: true,
		})
		udp := s.AddUDPEntrypoint(8080)

		services = append(services, s)

		s = deployment.Config().NewService("db", "postgres:14-alpine")
		tcp := s.AddTCPEntrypoint(5432)

		services = append(services, s)

		s = deployment.Config().NewService("cache", "redis:6-alpine")
		services = append(services, s)

		entrypoints := services.Entrypoints()

		assert.HasLength(t, 3, entrypoints)
		assert.Equal(t, http, entrypoints[0])
		assert.Equal(t, udp, entrypoints[1])
		assert.Equal(t, tcp, entrypoints[2])

		entrypoints = services.CustomEntrypoints()

		assert.HasLength(t, 2, entrypoints)
		assert.Equal(t, udp, entrypoints[0])
		assert.Equal(t, tcp, entrypoints[1])
	})

	t.Run("should implement the valuer interface", func(t *testing.T) {
		var services domain.Services
		app := fixture.App(fixture.WithAppName("my-app"))
		deployment := fixture.Deployment(fixture.FromApp(app))
		appidLower := strings.ToLower(string(deployment.ID().AppID()))

		s := deployment.Config().NewService("app", "")
		s.AddHttpEntrypoint(deployment.Config(), 80, domain.HttpEntrypointOptions{
			UseDefaultSubdomain: true,
			Managed:             true,
		})
		s.AddTCPEntrypoint(8080)

		services = append(services, s)

		s = deployment.Config().NewService("db", "postgres:14-alpine")
		s.AddTCPEntrypoint(5432)

		services = append(services, s)

		s = deployment.Config().NewService("cache", "redis:6-alpine")
		services = append(services, s)

		value, err := services.Value()

		assert.Nil(t, err)
		assert.Equal(t, fmt.Sprintf(`[{"name":"app","qualified_name":"my-app-production-%s-app","image":"my-app-%s/app:production","entrypoints":[{"name":"my-app-production-%s-app-80-http","is_custom":false,"router":"http","subdomain":"my-app","port":80},{"name":"my-app-production-%s-app-8080-tcp","is_custom":true,"router":"tcp","subdomain":null,"port":8080}]},{"name":"db","qualified_name":"my-app-production-%s-db","image":"postgres:14-alpine","entrypoints":[{"name":"my-app-production-%s-db-5432-tcp","is_custom":true,"router":"tcp","subdomain":null,"port":5432}]},{"name":"cache","qualified_name":"my-app-production-%s-cache","image":"redis:6-alpine","entrypoints":[]}]`,
			appidLower, appidLower, appidLower, appidLower, appidLower, appidLower, appidLower), value.(string))
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
		assert.Equal(t, `[{"name":"app","qualified_name":"my-app-production-2fa8domd2sh7ehyqlxf7jvj57xs-app","image":"my-app-2fa8domd2sh7ehyqlxf7jvj57xs/app:production","entrypoints":[{"name":"my-app-production-2fa8domd2sh7ehyqlxf7jvj57xs-app-80-http","is_custom":false,"router":"http","subdomain":"my-app","port":80},{"name":"my-app-production-2fa8domd2sh7ehyqlxf7jvj57xs-app-8080-tcp","is_custom":true,"router":"tcp","subdomain":null,"port":8080}]},{"name":"db","qualified_name":"my-app-production-2fa8domd2sh7ehyqlxf7jvj57xs-db","image":"postgres:14-alpine","entrypoints":[{"name":"my-app-production-2fa8domd2sh7ehyqlxf7jvj57xs-db-5432-tcp","is_custom":true,"router":"tcp","subdomain":null,"port":5432}]},{"name":"cache","qualified_name":"my-app-production-2fa8domd2sh7ehyqlxf7jvj57xs-cache","image":"redis:6-alpine","entrypoints":[]}]`, v.(string))
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
