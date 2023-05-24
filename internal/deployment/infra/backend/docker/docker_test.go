package docker_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	dockertypes "github.com/docker/docker/api/types"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/deployment/infra/backend/docker"
	"github.com/YuukanOO/seelf/internal/deployment/infra/trigger/raw"
	"github.com/YuukanOO/seelf/pkg/log"
	"github.com/YuukanOO/seelf/pkg/testutil"
	"github.com/compose-spec/compose-go/types"
	"github.com/docker/cli/cli/command"
	"github.com/docker/compose/v2/pkg/api"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

func Test_Run(t *testing.T) {
	t.Cleanup(func() {
		os.RemoveAll(dataDir)
	})

	backend := func(opts docker.Options) (docker.Backend, *composeMockService, *dockerCliMockService) {
		composeMock := &composeMockService{}
		dockerMock := newDockerMockService()
		return docker.New(opts, log.NewLogger(false), docker.WithDockerAndCompose(dockerMock, composeMock)), composeMock, dockerMock.cli
	}

	t.Run("should setup the balancer correctly without SSL", func(t *testing.T) {
		opts := options{domain: "http://docker.localhost"}
		dockerBackend, composeMock, _ := backend(opts)

		err := dockerBackend.Setup()

		testutil.IsNil(t, err)
		project := composeMock.project
		testutil.IsNotNil(t, project)
		testutil.Equals(t, "seelf-internal", project.Name)
		testutil.HasLength(t, project.Services, 1)
		testutil.Equals(t, "balancer", project.Services[0].Name)
		testutil.Equals(t, types.RestartPolicyUnlessStopped, project.Services[0].Restart)
		testutil.Equals(t, "traefik:v2.6", project.Services[0].Image)
		testutil.DeepEquals(t, []string{
			"--providers.docker",
			"--providers.docker.network=seelf-public",
			"--providers.docker.exposedbydefault=false",
		}, project.Services[0].Command)
		testutil.HasLength(t, project.Services[0].Ports, 1)
		testutil.Equals(t, "80", project.Services[0].Ports[0].Published)
		testutil.Equals(t, 80, project.Services[0].Ports[0].Target)
		testutil.HasLength(t, project.Services[0].Volumes, 1)
		testutil.Equals(t, "/var/run/docker.sock", project.Services[0].Volumes[0].Source)
		testutil.Equals(t, "/var/run/docker.sock", project.Services[0].Volumes[0].Target)

		testutil.Equals(t, 1, len(project.Networks))
		testutil.Equals(t, "seelf-public", project.Networks["default"].Name)
	})

	t.Run("should setup the balancer correctly with SSL", func(t *testing.T) {
		opts := options{"https://docker.localhost", "someone@example.com"}
		dockerBackend, composeMock, _ := backend(opts)

		err := dockerBackend.Setup()
		testutil.IsNil(t, err)
		project := composeMock.project
		testutil.IsNotNil(t, project)
		testutil.Equals(t, "seelf-internal", project.Name)
		testutil.HasLength(t, project.Services, 1)
		testutil.Equals(t, "balancer", project.Services[0].Name)
		testutil.Equals(t, types.RestartPolicyUnlessStopped, project.Services[0].Restart)
		testutil.Equals(t, "traefik:v2.6", project.Services[0].Image)
		testutil.DeepEquals(t, []string{
			"--providers.docker",
			"--providers.docker.network=seelf-public",
			"--providers.docker.exposedbydefault=false",
			"--entrypoints.web.address=:80",
			"--entrypoints.web.http.redirections.entryPoint.to=websecure",
			"--entrypoints.web.http.redirections.entryPoint.scheme=https",
			"--entrypoints.websecure.address=:443",
			"--certificatesresolvers.seelfresolver.acme.tlschallenge=true",
			"--certificatesresolvers.seelfresolver.acme.email=someone@example.com",
			"--certificatesresolvers.seelfresolver.acme.storage=/letsencrypt/acme.json",
		}, project.Services[0].Command)
		testutil.HasLength(t, project.Services[0].Ports, 2)
		testutil.Equals(t, "80", project.Services[0].Ports[0].Published)
		testutil.Equals(t, 80, project.Services[0].Ports[0].Target)
		testutil.Equals(t, "443", project.Services[0].Ports[1].Published)
		testutil.Equals(t, 443, project.Services[0].Ports[1].Target)
		testutil.HasLength(t, project.Services[0].Volumes, 2)
		testutil.Equals(t, "/var/run/docker.sock", project.Services[0].Volumes[0].Source)
		testutil.Equals(t, "/var/run/docker.sock", project.Services[0].Volumes[0].Target)
		testutil.Equals(t, "letsencrypt", project.Services[0].Volumes[1].Source)
		testutil.Equals(t, "/letsencrypt", project.Services[0].Volumes[1].Target)

		testutil.Equals(t, 1, len(project.Networks))
		testutil.Equals(t, "seelf-public", project.Networks["default"].Name)

		testutil.Equals(t, 1, len(project.Volumes))
		testutil.Equals(t, "seelf-internal_letsencrypt", project.Volumes["letsencrypt"].Name)
	})

	t.Run("should err if no compose file was found for a deployment", func(t *testing.T) {
		opts := options{domain: "http://docker.localhost"}
		app := domain.NewApp("my-app", "uid")
		depl, _ := app.NewDeployment(1, domain.NewMeta("test", ""), domain.Production, opts, "uid")
		dockerBackend, _, _ := backend(opts)

		_, err := dockerBackend.Run(context.Background(), depl)

		testutil.IsTrue(t, errors.Is(err, docker.ErrOpenComposeFileFailed))
	})

	testServices := func(t *testing.T, opts options) {
		app := domain.NewApp("my-app", "uid")

		dsn := "postgres://prodapp:passprod@db/app?sslmode=disable"
		postgresUser := "prodapp"
		postgresPassword := "passprod"

		app.HasEnvironmentVariables(domain.EnvironmentsEnv{
			domain.Production: domain.ServicesEnv{
				"app": domain.EnvVars{
					"DSN": dsn,
				},
				"db": domain.EnvVars{
					"POSTGRES_USER":     postgresUser,
					"POSTGRES_PASSWORD": postgresPassword,
				},
			},
		})

		trigger := raw.New(opts)
		meta, _ := trigger.Prepare(app, `
services:
  app:
    restart: unless-stopped
    build: .
    environment:
      - DSN=postgres://app:apppa55word@db/app?sslmode=disable
    depends_on:
      - db
    ports:
      - "8080:8080"
  sidecar:
    image: traefik/whoami
    ports:
      - "8889:80"
    profiles:
      - production
  stagingonly:
    image: traefik/whoami
    ports:
      - "8888:80"
    profiles:
      - staging
  db:
    restart: unless-stopped
    image: postgres:14-alpine
    volumes:
      - dbdata:/var/lib/postgresql/data
    environment:
      - POSTGRES_USER=app
      - POSTGRES_PASSWORD=apppa55word
volumes:
  dbdata:
`)
		depl, _ := app.NewDeployment(1, meta, domain.Production, opts, "uid")
		testutil.IsNil(t, trigger.Fetch(context.Background(), depl))

		dockerBackend, composeMock, cliMock := backend(opts)

		services, err := dockerBackend.Run(context.Background(), depl)

		testutil.IsNil(t, err)
		testutil.HasLength(t, services, 3)
		testutil.Equals(t, "app", services[0].Name())
		testutil.Equals(t, "my-app/app:production", services[0].Image())
		if opts.Domain().UseSSL() {
			testutil.Equals(t, "https://my-app.docker.localhost", services[0].Url().MustGet().String())
		} else {
			testutil.Equals(t, "http://my-app.docker.localhost", services[0].Url().MustGet().String())
		}

		testutil.Equals(t, "db", services[1].Name())
		testutil.Equals(t, "postgres:14-alpine", services[1].Image())
		testutil.IsFalse(t, services[1].Url().HasValue())

		testutil.Equals(t, "sidecar", services[2].Name())
		testutil.Equals(t, "traefik/whoami", services[2].Image())
		if opts.Domain().UseSSL() {
			testutil.Equals(t, "https://sidecar.my-app.docker.localhost", services[2].Url().MustGet().String())
		} else {
			testutil.Equals(t, "http://sidecar.my-app.docker.localhost", services[2].Url().MustGet().String())
		}

		project := composeMock.project
		testutil.IsNotNil(t, project)
		testutil.Equals(t, "my-app-production", project.Name)

		for _, service := range composeMock.project.Services {
			switch service.Name {
			case "app":
				testutil.Equals(t, "my-app/app:production", service.Image)
				testutil.Equals(t, types.RestartPolicyUnlessStopped, service.Restart)
				testutil.Equals(t, types.PullPolicyBuild, service.PullPolicy)
				testutil.DeepEquals(t, types.Labels{
					docker.AppLabel:         string(app.ID()),
					docker.EnvironmentLabel: string(domain.Production),
				}, service.Build.Labels)
				testutil.HasLength(t, service.Ports, 0)
				testutil.DeepEquals(t, types.MappingWithEquals{
					"DSN": &dsn,
				}, service.Environment)
				if opts.Domain().UseSSL() {
					testutil.DeepEquals(t, types.Labels{
						docker.AppLabel:         string(app.ID()),
						docker.EnvironmentLabel: string(domain.Production),
						"traefik.enable":        "true",
						"traefik.http.routers.my-app-production-app.rule":             "Host(`my-app.docker.localhost`)",
						"traefik.http.routers.my-app-production-app.tls.certresolver": "seelfresolver",
					}, service.Labels)
				} else {
					testutil.DeepEquals(t, types.Labels{
						docker.AppLabel:         string(app.ID()),
						docker.EnvironmentLabel: string(domain.Production),
						"traefik.enable":        "true",
						"traefik.http.routers.my-app-production-app.rule": "Host(`my-app.docker.localhost`)",
					}, service.Labels)
				}
			case "db":
				testutil.Equals(t, "postgres:14-alpine", service.Image)
				testutil.Equals(t, types.RestartPolicyUnlessStopped, service.Restart)
				testutil.DeepEquals(t, types.Labels{
					docker.AppLabel:         string(app.ID()),
					docker.EnvironmentLabel: string(domain.Production),
				}, service.Labels)
				testutil.DeepEquals(t, types.MappingWithEquals{
					"POSTGRES_USER":     &postgresUser,
					"POSTGRES_PASSWORD": &postgresPassword,
				}, service.Environment)
				testutil.HasLength(t, service.Volumes, 1)
				testutil.Equals(t, "dbdata", service.Volumes[0].Source)
				testutil.Equals(t, "/var/lib/postgresql/data", service.Volumes[0].Target)
			case "sidecar":
				testutil.Equals(t, "traefik/whoami", service.Image)
				testutil.HasLength(t, service.Ports, 0)
				testutil.DeepEquals(t, types.MappingWithEquals{}, service.Environment)
				if opts.Domain().UseSSL() {
					testutil.DeepEquals(t, types.Labels{
						docker.AppLabel:         string(app.ID()),
						docker.EnvironmentLabel: string(domain.Production),
						"traefik.enable":        "true",
						"traefik.http.routers.my-app-production-sidecar.rule":             "Host(`sidecar.my-app.docker.localhost`)",
						"traefik.http.routers.my-app-production-sidecar.tls.certresolver": "seelfresolver",
					}, service.Labels)
				} else {
					testutil.DeepEquals(t, types.Labels{
						docker.AppLabel:         string(app.ID()),
						docker.EnvironmentLabel: string(domain.Production),
						"traefik.enable":        "true",
						"traefik.http.routers.my-app-production-sidecar.rule": "Host(`sidecar.my-app.docker.localhost`)",
					}, service.Labels)
				}
			default:
				t.Fatal("no other service expected")
			}
		}

		testutil.Equals(t, 2, len(project.Networks))
		testutil.Equals(t, "my-app-production_default", project.Networks["default"].Name)
		testutil.DeepEquals(t, types.Labels{
			docker.AppLabel:         string(app.ID()),
			docker.EnvironmentLabel: string(domain.Production),
		}, composeMock.project.Networks["default"].Labels)
		testutil.Equals(t, "seelf-public", project.Networks["seelf-public"].Name)
		testutil.Equals(t, 0, len(project.Networks["seelf-public"].Labels))
		testutil.IsTrue(t, project.Networks["seelf-public"].External.External)

		testutil.Equals(t, 1, len(project.Volumes))
		testutil.Equals(t, "my-app-production_dbdata", project.Volumes["dbdata"].Name)
		testutil.DeepEquals(t, types.Labels{
			docker.AppLabel:         string(app.ID()),
			docker.EnvironmentLabel: string(domain.Production),
		}, project.Volumes["dbdata"].Labels)

		testutil.DeepEquals(t, filters.NewArgs(
			filters.Arg("dangling", "true"),
			filters.Arg("label", fmt.Sprintf("%s=%s", docker.AppLabel, app.ID())),
			filters.Arg("label", fmt.Sprintf("%s=%s", docker.EnvironmentLabel, domain.Production)),
		), cliMock.pruneFilter)
	}

	t.Run("should correctly expose services from a compose file without SSL", func(t *testing.T) {
		opts := options{domain: "http://docker.localhost"}
		testServices(t, opts)
	})

	t.Run("should correctly expose services from a compose file with SSL", func(t *testing.T) {
		opts := options{domain: "https://docker.localhost", email: "someone@example.com"}
		testServices(t, opts)
	})
}

type composeMockService struct {
	api.Service
	project *types.Project
	options api.UpOptions
}

func (c *composeMockService) Up(ctx context.Context, project *types.Project, options api.UpOptions) error {
	c.project = project
	c.options = options
	return nil
}

type (
	dockerMockService struct {
		command.Cli
		cli *dockerCliMockService
	}

	dockerCliMockService struct {
		client.APIClient
		pruneFilter filters.Args
	}
)

func newDockerMockService() *dockerMockService {
	return &dockerMockService{
		cli: &dockerCliMockService{},
	}
}

func (d *dockerMockService) Client() client.APIClient {
	return d.cli
}

func (d *dockerMockService) Apply(ops ...command.DockerCliOption) error {
	return nil
}

func (d *dockerCliMockService) ImagesPrune(ctx context.Context, pruneFilter filters.Args) (dockertypes.ImagesPruneReport, error) {
	d.pruneFilter = pruneFilter
	return dockertypes.ImagesPruneReport{}, nil
}

const dataDir = "test_data"

type options struct {
	domain string
	email  string
}

func (options) DataDir() string   { return dataDir }
func (o options) AppsDir() string { return filepath.Join(o.DataDir(), "apps") }
func (o options) LogsDir() string { return filepath.Join(o.DataDir(), "logs") }
func (o options) Domain() domain.Url {
	d, _ := domain.UrlFrom(o.domain)
	return d
}
func (o options) AcmeEmail() string { return o.email }
func (o options) Execute(d domain.DeploymentTemplateData) string {
	return filepath.Join(strconv.Itoa(int(d.Number)), string(d.Environment))
}
