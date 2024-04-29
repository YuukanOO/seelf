package domain_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/must"
	"github.com/YuukanOO/seelf/pkg/testutil"
)

func Test_Service(t *testing.T) {
	app := must.Panic(domain.NewApp("my-app",
		domain.NewEnvironmentConfigRequirement(domain.NewEnvironmentConfig("production-target"), true, true),
		domain.NewEnvironmentConfigRequirement(domain.NewEnvironmentConfig("staging-target"), true, true),
		"uid"))
	appidLower := strings.ToLower(string(app.ID()))
	config := must.Panic(app.ConfigSnapshotFor(domain.Production))

	t.Run("could be created from a deployment configuration", func(t *testing.T) {
		s := config.NewService("db", "postgres:14-alpine")

		testutil.Equals(t, "db", s.Name())
		testutil.Equals(t, "postgres:14-alpine", s.Image())

		s = config.NewService("app", "")

		testutil.Equals(t, "app", s.Name())
		testutil.Equals(t, fmt.Sprintf("my-app-%s/app:production", appidLower), s.Image())
	})

	t.Run("should populate the subdomain when adding HTTP entrypoints", func(t *testing.T) {
		s := config.NewService("app", "")

		e := s.AddHttpEntrypoint(config, 80, domain.HttpEntrypointOptions{
			Managed:             true,
			UseDefaultSubdomain: true,
		})
		testutil.Equals(t, fmt.Sprintf("my-app-production-%s-app-80-http", appidLower), string(e.Name()))
		testutil.Equals(t, domain.RouterHttp, e.Router())
		testutil.IsFalse(t, e.IsCustom())
		testutil.Equals(t, "my-app", e.Subdomain().Get(""))
		testutil.Equals(t, 80, e.Port())

		e = s.AddHttpEntrypoint(config, 8080, domain.HttpEntrypointOptions{})
		testutil.Equals(t, fmt.Sprintf("my-app-production-%s-app-8080-http", appidLower), string(e.Name()))
		testutil.Equals(t, domain.RouterHttp, e.Router())
		testutil.IsTrue(t, e.IsCustom())
		testutil.Equals(t, "my-app", e.Subdomain().Get(""))
		testutil.Equals(t, 8080, e.Port())

		same := s.AddHttpEntrypoint(config, 8080, domain.HttpEntrypointOptions{})
		testutil.Equals(t, e, same)
	})

	t.Run("could have one or more TCP/UDP entrypoints attached", func(t *testing.T) {
		s := config.NewService("app", "")

		tcp := s.AddTCPEntrypoint(8080)
		testutil.Equals(t, fmt.Sprintf("my-app-production-%s-app-8080-tcp", appidLower), string(tcp.Name()))
		testutil.Equals(t, domain.RouterTcp, tcp.Router())
		testutil.IsTrue(t, tcp.IsCustom())
		testutil.IsFalse(t, tcp.Subdomain().HasValue())
		testutil.Equals(t, 8080, tcp.Port())

		udp := s.AddUDPEntrypoint(8080)
		testutil.Equals(t, fmt.Sprintf("my-app-production-%s-app-8080-udp", appidLower), string(udp.Name()))
		testutil.Equals(t, domain.RouterUdp, udp.Router())
		testutil.IsTrue(t, udp.IsCustom())
		testutil.IsFalse(t, udp.Subdomain().HasValue())
		testutil.Equals(t, 8080, udp.Port())

		same := s.AddTCPEntrypoint(8080)
		testutil.Equals(t, tcp, same)

		same = s.AddUDPEntrypoint(8080)
		testutil.Equals(t, udp, same)
	})
}

func Test_Services(t *testing.T) {
	app := must.Panic(domain.NewApp("my-app",
		domain.NewEnvironmentConfigRequirement(domain.NewEnvironmentConfig("production-target"), true, true),
		domain.NewEnvironmentConfigRequirement(domain.NewEnvironmentConfig("staging-target"), true, true),
		"uid"))
	appidLower := strings.ToLower(string(app.ID()))
	config := must.Panic(app.ConfigSnapshotFor(domain.Production))

	t.Run("should be able to return all entrypoints", func(t *testing.T) {
		var services domain.Services

		s := config.NewService("app", "")
		http := s.AddHttpEntrypoint(config, 80, domain.HttpEntrypointOptions{
			Managed: true,
		})
		udp := s.AddUDPEntrypoint(8080)

		services = append(services, s)

		s = config.NewService("db", "postgres:14-alpine")
		tcp := s.AddTCPEntrypoint(5432)

		services = append(services, s)

		s = config.NewService("cache", "redis:6-alpine")
		services = append(services, s)

		entrypoints := services.Entrypoints()

		testutil.HasLength(t, entrypoints, 3)
		testutil.Equals(t, http, entrypoints[0])
		testutil.Equals(t, udp, entrypoints[1])
		testutil.Equals(t, tcp, entrypoints[2])

		entrypoints = services.CustomEntrypoints()

		testutil.HasLength(t, entrypoints, 2)
		testutil.Equals(t, udp, entrypoints[0])
		testutil.Equals(t, tcp, entrypoints[1])
	})

	t.Run("should implement the valuer interface", func(t *testing.T) {
		var services domain.Services

		s := config.NewService("app", "")
		s.AddHttpEntrypoint(config, 80, domain.HttpEntrypointOptions{
			UseDefaultSubdomain: true,
			Managed:             true,
		})
		s.AddTCPEntrypoint(8080)

		services = append(services, s)

		s = config.NewService("db", "postgres:14-alpine")
		s.AddTCPEntrypoint(5432)

		services = append(services, s)

		s = config.NewService("cache", "redis:6-alpine")
		services = append(services, s)

		value, err := services.Value()

		testutil.IsNil(t, err)
		testutil.Equals(t, fmt.Sprintf(`[{"name":"app","qualified_name":"my-app-production-%s-app","image":"my-app-%s/app:production","entrypoints":[{"name":"my-app-production-%s-app-80-http","is_custom":false,"router":"http","subdomain":"my-app","port":80},{"name":"my-app-production-%s-app-8080-tcp","is_custom":true,"router":"tcp","subdomain":null,"port":8080}]},{"name":"db","qualified_name":"my-app-production-%s-db","image":"postgres:14-alpine","entrypoints":[{"name":"my-app-production-%s-db-5432-tcp","is_custom":true,"router":"tcp","subdomain":null,"port":5432}]},{"name":"cache","qualified_name":"my-app-production-%s-cache","image":"redis:6-alpine","entrypoints":[]}]`,
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

		testutil.IsNil(t, err)
		testutil.HasLength(t, services, 3)

		v, err := services.Value()

		testutil.IsNil(t, err)
		testutil.Equals(t, `[{"name":"app","qualified_name":"my-app-production-2fa8domd2sh7ehyqlxf7jvj57xs-app","image":"my-app-2fa8domd2sh7ehyqlxf7jvj57xs/app:production","entrypoints":[{"name":"my-app-production-2fa8domd2sh7ehyqlxf7jvj57xs-app-80-http","is_custom":false,"router":"http","subdomain":"my-app","port":80},{"name":"my-app-production-2fa8domd2sh7ehyqlxf7jvj57xs-app-8080-tcp","is_custom":true,"router":"tcp","subdomain":null,"port":8080}]},{"name":"db","qualified_name":"my-app-production-2fa8domd2sh7ehyqlxf7jvj57xs-db","image":"postgres:14-alpine","entrypoints":[{"name":"my-app-production-2fa8domd2sh7ehyqlxf7jvj57xs-db-5432-tcp","is_custom":true,"router":"tcp","subdomain":null,"port":5432}]},{"name":"cache","qualified_name":"my-app-production-2fa8domd2sh7ehyqlxf7jvj57xs-cache","image":"redis:6-alpine","entrypoints":[]}]`, v.(string))
	})
}

func Test_Port(t *testing.T) {
	t.Run("should be able to parse a port from a raw string value", func(t *testing.T) {
		_, err := domain.ParsePort("failed")

		testutil.ErrorIs(t, domain.ErrInvalidPort, err)

		p, err := domain.ParsePort("8080")
		testutil.IsNil(t, err)
		testutil.Equals(t, 8080, p)
	})

	t.Run("should convert the port to a string", func(t *testing.T) {
		p := domain.Port(8080)
		testutil.Equals(t, "8080", p.String())
	})

	t.Run("should convert the port to a uint32", func(t *testing.T) {
		p := domain.Port(8080)
		testutil.Equals(t, 8080, p.Uint32())
	})
}

func Test_EntrypointName(t *testing.T) {
	t.Run("should provide a protocol", func(t *testing.T) {
		testutil.Equals(t, "tcp", domain.EntrypointName("my-app-production-2fa8domd2sh7ehyqlxf7jvj57xs-app-8080-http").Protocol())
		testutil.Equals(t, "tcp", domain.EntrypointName("my-app-production-2fa8domd2sh7ehyqlxf7jvj57xs-app-8080-tcp").Protocol())
		testutil.Equals(t, "udp", domain.EntrypointName("my-app-production-2fa8domd2sh7ehyqlxf7jvj57xs-app-8080-udp").Protocol())
	})
}
