package docker_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	dockertypes "github.com/docker/docker/api/types"

	"github.com/YuukanOO/seelf/cmd/config"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/deployment/infra/artifact"
	"github.com/YuukanOO/seelf/internal/deployment/infra/provider/docker"
	"github.com/YuukanOO/seelf/internal/deployment/infra/source/raw"
	"github.com/YuukanOO/seelf/pkg/log"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/must"
	"github.com/YuukanOO/seelf/pkg/ssh"
	"github.com/YuukanOO/seelf/pkg/testutil"
	"github.com/compose-spec/compose-go/types"
	"github.com/docker/cli/cli/command"
	"github.com/docker/compose/v2/pkg/api"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

type options interface {
	docker.Options
	artifact.LocalOptions
}

func Test_Docker(t *testing.T) {
	logger := must.Panic(log.NewLogger())
	targetWithSSL := must.Panic(domain.NewTarget("my-target",
		must.Panic(domain.UrlFrom("https://docker.localhost")), true, docker.Data{}, true, "uid"))
	targetWithoutSSL := must.Panic(domain.NewTarget("my-target",
		must.Panic(domain.UrlFrom("http://docker.localhost")), true, docker.Data{}, true, "uid"))

	composeMock := &composeMockService{}
	dockerMock := newDockerMockService()

	sut := func(opts options) (domain.Provider, domain.ArtifactManager, *composeMockService, *dockerCliMockService) {
		artifactManager := artifact.NewLocal(opts, logger)

		t.Cleanup(func() {
			os.RemoveAll(opts.DataDir())
		})

		return docker.New(opts, logger, docker.WithDockerAndCompose(dockerMock, composeMock)), artifactManager, composeMock, dockerMock.cli
	}

	t.Run("should be able to prepare a docker provider config from raw payload", func(t *testing.T) {
		tests := []struct {
			payload  docker.Body
			expected docker.Data
			existing []domain.ProviderConfig
		}{
			{
				payload:  docker.Body{},
				expected: docker.Data{},
			},
			{
				payload: docker.Body{
					Host: monad.Value("localhost"),
					User: monad.Value("test"),
					Port: monad.Value(2222),
				},
				expected: docker.Data{
					Host: monad.Value[ssh.Host]("localhost"),
					User: monad.Value("test"),
					Port: monad.Value(2222),
				},
			},
			{
				payload: docker.Body{
					Host: monad.Value("localhost"),
					User: monad.Value("test"),
					Port: monad.Value(2222),
					PrivateKey: monad.PatchValue(`-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAlwAAAAdzc2gtcn
NhAAAAAwEAAQAAAIEAwa48yfWFi3uIdqzuf9X7C2Zxfea/Iaaw0zIwHudpF8U92WVIiC5l
oEuW1+OaVi3UWfIEjWMV1tHGysrHOwtwc34BPCJqJknUQO/KtDTBTJ4Pryhw1bWPC999Lz
a+yrCTdNQYBzoROXKExZgPFh9pTMi5wqpHDuOQ2qZFIEI3lT0AAAIQWL0H31i9B98AAAAH
c3NoLXJzYQAAAIEAwa48yfWFi3uIdqzuf9X7C2Zxfea/Iaaw0zIwHudpF8U92WVIiC5loE
uW1+OaVi3UWfIEjWMV1tHGysrHOwtwc34BPCJqJknUQO/KtDTBTJ4Pryhw1bWPC999Lza+
yrCTdNQYBzoROXKExZgPFh9pTMi5wqpHDuOQ2qZFIEI3lT0AAAADAQABAAAAgCThyTGsT4
IARDxVMhWl6eiB2ZrgFgWSeJm/NOqtppWgOebsIqPMMg4UVuVFsl422/lE3RkPhVkjGXgE
pWvZAdCnmLmApK8wK12vF334lZhZT7t3Z9EzJps88PWEHo7kguf285HcnUM7FlFeissJdk
kXly34y7/3X/a6Tclm+iABAAAAQE0xR/KxZ39slwfMv64Rz7WKk1PPskaryI29aHE3mKHk
pY2QA+P3QlrKxT/VWUMjHUbNNdYfJm48xu0SGNMRdKMAAABBAORh2NP/06JUV3J9W/2Hju
X1ViJuqqcQnJPVzpgSL826EC2xwOECTqoY8uvFpUdD7CtpksIxNVqRIhuNOlz0lqEAAABB
ANkaHTTaPojClO0dKJ/Zjs7pWOCGliebBYprQ/Y4r9QLBkC/XaWMS26gFIrjgC7D2Rv+rZ
wSD0v0RcmkITP1ZR0AAAAYcHF1ZXJuYUBMdWNreUh5ZHJvLmxvY2FsAQID
-----END OPENSSH PRIVATE KEY-----`),
				},
				expected: docker.Data{
					Host: monad.Value[ssh.Host]("localhost"),
					User: monad.Value("test"),
					Port: monad.Value(2222),
					PrivateKey: monad.Value[ssh.PrivateKey](`-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAlwAAAAdzc2gtcn
NhAAAAAwEAAQAAAIEAwa48yfWFi3uIdqzuf9X7C2Zxfea/Iaaw0zIwHudpF8U92WVIiC5l
oEuW1+OaVi3UWfIEjWMV1tHGysrHOwtwc34BPCJqJknUQO/KtDTBTJ4Pryhw1bWPC999Lz
a+yrCTdNQYBzoROXKExZgPFh9pTMi5wqpHDuOQ2qZFIEI3lT0AAAIQWL0H31i9B98AAAAH
c3NoLXJzYQAAAIEAwa48yfWFi3uIdqzuf9X7C2Zxfea/Iaaw0zIwHudpF8U92WVIiC5loE
uW1+OaVi3UWfIEjWMV1tHGysrHOwtwc34BPCJqJknUQO/KtDTBTJ4Pryhw1bWPC999Lza+
yrCTdNQYBzoROXKExZgPFh9pTMi5wqpHDuOQ2qZFIEI3lT0AAAADAQABAAAAgCThyTGsT4
IARDxVMhWl6eiB2ZrgFgWSeJm/NOqtppWgOebsIqPMMg4UVuVFsl422/lE3RkPhVkjGXgE
pWvZAdCnmLmApK8wK12vF334lZhZT7t3Z9EzJps88PWEHo7kguf285HcnUM7FlFeissJdk
kXly34y7/3X/a6Tclm+iABAAAAQE0xR/KxZ39slwfMv64Rz7WKk1PPskaryI29aHE3mKHk
pY2QA+P3QlrKxT/VWUMjHUbNNdYfJm48xu0SGNMRdKMAAABBAORh2NP/06JUV3J9W/2Hju
X1ViJuqqcQnJPVzpgSL826EC2xwOECTqoY8uvFpUdD7CtpksIxNVqRIhuNOlz0lqEAAABB
ANkaHTTaPojClO0dKJ/Zjs7pWOCGliebBYprQ/Y4r9QLBkC/XaWMS26gFIrjgC7D2Rv+rZ
wSD0v0RcmkITP1ZR0AAAAYcHF1ZXJuYUBMdWNreUh5ZHJvLmxvY2FsAQID
-----END OPENSSH PRIVATE KEY-----`),
				},
			},
			{
				payload: docker.Body{
					Host: monad.Value("localhost"),
					User: monad.Value("test"),
					Port: monad.Value(2222),
					PrivateKey: monad.PatchValue(`-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAlwAAAAdzc2gtcn
NhAAAAAwEAAQAAAIEAwa48yfWFi3uIdqzuf9X7C2Zxfea/Iaaw0zIwHudpF8U92WVIiC5l
oEuW1+OaVi3UWfIEjWMV1tHGysrHOwtwc34BPCJqJknUQO/KtDTBTJ4Pryhw1bWPC999Lz
a+yrCTdNQYBzoROXKExZgPFh9pTMi5wqpHDuOQ2qZFIEI3lT0AAAIQWL0H31i9B98AAAAH
c3NoLXJzYQAAAIEAwa48yfWFi3uIdqzuf9X7C2Zxfea/Iaaw0zIwHudpF8U92WVIiC5loE
uW1+OaVi3UWfIEjWMV1tHGysrHOwtwc34BPCJqJknUQO/KtDTBTJ4Pryhw1bWPC999Lza+
yrCTdNQYBzoROXKExZgPFh9pTMi5wqpHDuOQ2qZFIEI3lT0AAAADAQABAAAAgCThyTGsT4
IARDxVMhWl6eiB2ZrgFgWSeJm/NOqtppWgOebsIqPMMg4UVuVFsl422/lE3RkPhVkjGXgE
pWvZAdCnmLmApK8wK12vF334lZhZT7t3Z9EzJps88PWEHo7kguf285HcnUM7FlFeissJdk
kXly34y7/3X/a6Tclm+iABAAAAQE0xR/KxZ39slwfMv64Rz7WKk1PPskaryI29aHE3mKHk
pY2QA+P3QlrKxT/VWUMjHUbNNdYfJm48xu0SGNMRdKMAAABBAORh2NP/06JUV3J9W/2Hju
X1ViJuqqcQnJPVzpgSL826EC2xwOECTqoY8uvFpUdD7CtpksIxNVqRIhuNOlz0lqEAAABB
ANkaHTTaPojClO0dKJ/Zjs7pWOCGliebBYprQ/Y4r9QLBkC/XaWMS26gFIrjgC7D2Rv+rZ
wSD0v0RcmkITP1ZR0AAAAYcHF1ZXJuYUBMdWNreUh5ZHJvLmxvY2FsAQID
-----END OPENSSH PRIVATE KEY-----`),
				},
				expected: docker.Data{
					Host: monad.Value[ssh.Host]("localhost"),
					User: monad.Value("test"),
					Port: monad.Value(2222),
					PrivateKey: monad.Value[ssh.PrivateKey](`-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAlwAAAAdzc2gtcn
NhAAAAAwEAAQAAAIEAwa48yfWFi3uIdqzuf9X7C2Zxfea/Iaaw0zIwHudpF8U92WVIiC5l
oEuW1+OaVi3UWfIEjWMV1tHGysrHOwtwc34BPCJqJknUQO/KtDTBTJ4Pryhw1bWPC999Lz
a+yrCTdNQYBzoROXKExZgPFh9pTMi5wqpHDuOQ2qZFIEI3lT0AAAIQWL0H31i9B98AAAAH
c3NoLXJzYQAAAIEAwa48yfWFi3uIdqzuf9X7C2Zxfea/Iaaw0zIwHudpF8U92WVIiC5loE
uW1+OaVi3UWfIEjWMV1tHGysrHOwtwc34BPCJqJknUQO/KtDTBTJ4Pryhw1bWPC999Lza+
yrCTdNQYBzoROXKExZgPFh9pTMi5wqpHDuOQ2qZFIEI3lT0AAAADAQABAAAAgCThyTGsT4
IARDxVMhWl6eiB2ZrgFgWSeJm/NOqtppWgOebsIqPMMg4UVuVFsl422/lE3RkPhVkjGXgE
pWvZAdCnmLmApK8wK12vF334lZhZT7t3Z9EzJps88PWEHo7kguf285HcnUM7FlFeissJdk
kXly34y7/3X/a6Tclm+iABAAAAQE0xR/KxZ39slwfMv64Rz7WKk1PPskaryI29aHE3mKHk
pY2QA+P3QlrKxT/VWUMjHUbNNdYfJm48xu0SGNMRdKMAAABBAORh2NP/06JUV3J9W/2Hju
X1ViJuqqcQnJPVzpgSL826EC2xwOECTqoY8uvFpUdD7CtpksIxNVqRIhuNOlz0lqEAAABB
ANkaHTTaPojClO0dKJ/Zjs7pWOCGliebBYprQ/Y4r9QLBkC/XaWMS26gFIrjgC7D2Rv+rZ
wSD0v0RcmkITP1ZR0AAAAYcHF1ZXJuYUBMdWNreUh5ZHJvLmxvY2FsAQID
-----END OPENSSH PRIVATE KEY-----`),
				},
				existing: []domain.ProviderConfig{
					docker.Data{
						Host:       monad.Value[ssh.Host]("other"),
						PrivateKey: monad.Value[ssh.PrivateKey]("other"),
					},
				},
			},
			{
				payload: docker.Body{
					Host: monad.Value("localhost"),
					User: monad.Value("test"),
					Port: monad.Value(2222),
				},
				expected: docker.Data{
					Host:       monad.Value[ssh.Host]("localhost"),
					User:       monad.Value("test"),
					Port:       monad.Value(2222),
					PrivateKey: monad.Value[ssh.PrivateKey]("other"),
				},
				existing: []domain.ProviderConfig{
					docker.Data{
						Host:       monad.Value[ssh.Host]("other"),
						PrivateKey: monad.Value[ssh.PrivateKey]("other"),
					},
				},
			},
			{
				payload: docker.Body{
					Host:       monad.Value("localhost"),
					PrivateKey: monad.Nil[string](),
				},
				expected: docker.Data{
					Host: monad.Value[ssh.Host]("localhost"),
					User: monad.Value("docker"),
					Port: monad.Value(22),
				},
				existing: []domain.ProviderConfig{
					docker.Data{
						Host:       monad.Value[ssh.Host]("other"),
						PrivateKey: monad.Value[ssh.PrivateKey]("other"),
					},
				},
			},
		}

		provider, _, _, _ := sut(config.Default(config.WithTestDefaults()))

		for _, tt := range tests {
			t.Run(fmt.Sprintf("%v", tt.payload), func(t *testing.T) {
				data, err := provider.Prepare(context.Background(), tt.payload, tt.existing...)

				testutil.IsNil(t, err)
				testutil.IsTrue(t, tt.expected == data)
			})
		}
	})

	// t.Run("should setup the balancer correctly without SSL", func(t *testing.T) {
	// 	opts := config.Default(config.WithTestDefaults())
	// 	provider, _, composeMock, _ := sut(opts)

	// 	err := provider.Setup()

	// 	testutil.IsNil(t, err)
	// 	project := composeMock.project
	// 	testutil.IsNotNil(t, project)
	// 	testutil.Equals(t, "seelf-internal", project.Name)
	// 	testutil.HasLength(t, project.Services, 1)
	// 	testutil.Equals(t, "balancer", project.Services[0].Name)
	// 	testutil.Equals(t, types.RestartPolicyUnlessStopped, project.Services[0].Restart)
	// 	testutil.Equals(t, "traefik:v2.6", project.Services[0].Image)
	// 	testutil.DeepEquals(t, []string{
	// 		"--providers.docker",
	// 		"--providers.docker.network=seelf-public",
	// 		"--providers.docker.exposedbydefault=false",
	// 	}, project.Services[0].Command)
	// 	testutil.HasLength(t, project.Services[0].Ports, 1)
	// 	testutil.Equals(t, "80", project.Services[0].Ports[0].Published)
	// 	testutil.Equals(t, 80, project.Services[0].Ports[0].Target)
	// 	testutil.HasLength(t, project.Services[0].Volumes, 1)
	// 	testutil.Equals(t, "/var/run/docker.sock", project.Services[0].Volumes[0].Source)
	// 	testutil.Equals(t, "/var/run/docker.sock", project.Services[0].Volumes[0].Target)

	// 	testutil.Equals(t, 1, len(project.Networks))
	// 	testutil.Equals(t, "seelf-public", project.Networks["default"].Name)
	// })

	// t.Run("should setup the balancer correctly with SSL", func(t *testing.T) {
	// 	opts := config.Default(
	// 		config.WithTestDefaults(),
	// 		config.WithBalancer("https://docker.localhost", "someone@example.com"),
	// 	)
	// 	provider, _, composeMock, _ := sut(opts)

	// 	err := provider.Setup()
	// 	testutil.IsNil(t, err)
	// 	project := composeMock.project
	// 	testutil.IsNotNil(t, project)
	// 	testutil.Equals(t, "seelf-internal", project.Name)
	// 	testutil.HasLength(t, project.Services, 1)
	// 	testutil.Equals(t, "balancer", project.Services[0].Name)
	// 	testutil.Equals(t, types.RestartPolicyUnlessStopped, project.Services[0].Restart)
	// 	testutil.Equals(t, "traefik:v2.6", project.Services[0].Image)
	// 	testutil.DeepEquals(t, []string{
	// 		"--providers.docker",
	// 		"--providers.docker.network=seelf-public",
	// 		"--providers.docker.exposedbydefault=false",
	// 		"--entrypoints.web.address=:80",
	// 		"--entrypoints.web.http.redirections.entryPoint.to=websecure",
	// 		"--entrypoints.web.http.redirections.entryPoint.scheme=https",
	// 		"--entrypoints.websecure.address=:443",
	// 		"--certificatesresolvers.seelfresolver.acme.tlschallenge=true",
	// 		"--certificatesresolvers.seelfresolver.acme.email=someone@example.com",
	// 		"--certificatesresolvers.seelfresolver.acme.storage=/letsencrypt/acme.json",
	// 	}, project.Services[0].Command)
	// 	testutil.HasLength(t, project.Services[0].Ports, 2)
	// 	testutil.Equals(t, "80", project.Services[0].Ports[0].Published)
	// 	testutil.Equals(t, 80, project.Services[0].Ports[0].Target)
	// 	testutil.Equals(t, "443", project.Services[0].Ports[1].Published)
	// 	testutil.Equals(t, 443, project.Services[0].Ports[1].Target)
	// 	testutil.HasLength(t, project.Services[0].Volumes, 2)
	// 	testutil.Equals(t, "/var/run/docker.sock", project.Services[0].Volumes[0].Source)
	// 	testutil.Equals(t, "/var/run/docker.sock", project.Services[0].Volumes[0].Target)
	// 	testutil.Equals(t, "letsencrypt", project.Services[0].Volumes[1].Source)
	// 	testutil.Equals(t, "/letsencrypt", project.Services[0].Volumes[1].Target)

	// 	testutil.Equals(t, 1, len(project.Networks))
	// 	testutil.Equals(t, "seelf-public", project.Networks["default"].Name)

	// 	testutil.Equals(t, 1, len(project.Volumes))
	// 	testutil.Equals(t, "seelf-internal_letsencrypt", project.Volumes["letsencrypt"].Name)
	// })

	t.Run("should err if no compose file was found for a deployment", func(t *testing.T) {
		opts := config.Default(config.WithTestDefaults())
		app := must.Panic(domain.NewApp("my-app", domain.NewEnvironmentConfig("1"), domain.NewEnvironmentConfig("1"), domain.AppNamingProductionAvailable|domain.AppNamingStagingAvailable, "uid"))
		depl := must.Panic(app.NewDeployment(1, raw.Data(""), domain.Production, "uid"))
		provider, artifactManager, _, _ := sut(opts)

		ctx := context.Background()
		deplCtx, err := artifactManager.PrepareBuild(ctx, depl)

		testutil.IsNil(t, err)

		defer deplCtx.Logger().Close()

		_, err = provider.Run(ctx, deplCtx, depl, targetWithoutSSL)

		testutil.IsTrue(t, errors.Is(err, docker.ErrOpenComposeFileFailed))
	})

	testServices := func(t *testing.T, opts options, target domain.Target) {
		provider, artifactManager, composeMock, cliMock := sut(opts)

		dsn := "postgres://prodapp:passprod@db/app?sslmode=disable"
		postgresUser := "prodapp"
		postgresPassword := "passprod"

		config := domain.NewEnvironmentConfig(target.ID())
		config.HasEnvironmentVariables(domain.ServicesEnv{
			"app": domain.EnvVars{
				"DSN": dsn,
			},
			"db": domain.EnvVars{
				"POSTGRES_USER":     postgresUser,
				"POSTGRES_PASSWORD": postgresPassword,
			},
		})

		app := must.Panic(domain.NewApp(
			"my-app",
			config,
			domain.NewEnvironmentConfig(target.ID()),
			domain.AppNamingProductionAvailable|domain.AppNamingStagingAvailable,
			"uid",
		))
		ctx := context.Background()
		src := raw.New()
		meta := must.Panic(src.Prepare(ctx, app, `
services:
  app:
    restart: unless-stopped
    build: .
    environment:
      - DSN=postgres://app:apppa55word@db/app?sslmode=disable
    depends_on:
      - db
    ports:
      - "8080"
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
`))

		depl := must.Panic(app.NewDeployment(1, meta, domain.Production, "uid"))
		deplCtx, err := artifactManager.PrepareBuild(ctx, depl)

		testutil.IsNil(t, err)

		defer deplCtx.Logger().Close()

		testutil.IsNil(t, src.Fetch(ctx, deplCtx, depl))

		services, err := provider.Run(ctx, deplCtx, depl, target)

		testutil.IsNil(t, err)
		testutil.HasLength(t, services, 3)
		testutil.Equals(t, "app", services[0].Name())
		testutil.Equals(t, "my-app/app:production", services[0].Image())
		if target.Domain().UseSSL() {
			testutil.Equals(t, "https://my-app.docker.localhost", services[0].Url().MustGet().String())
		} else {
			testutil.Equals(t, "http://my-app.docker.localhost", services[0].Url().MustGet().String())
		}

		testutil.Equals(t, "db", services[1].Name())
		testutil.Equals(t, "postgres:14-alpine", services[1].Image())
		testutil.IsFalse(t, services[1].Url().HasValue())

		testutil.Equals(t, "sidecar", services[2].Name())
		testutil.Equals(t, "traefik/whoami", services[2].Image())
		if target.Domain().UseSSL() {
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
				if target.Domain().UseSSL() {
					testutil.DeepEquals(t, types.Labels{
						docker.AppLabel:         string(app.ID()),
						docker.EnvironmentLabel: string(domain.Production),
						"traefik.enable":        "true",
						"traefik.http.services.my-app-production-app.loadbalancer.server.port": "8080",
						"traefik.http.routers.my-app-production-app.rule":                      "Host(`my-app.docker.localhost`)",
						"traefik.http.routers.my-app-production-app.tls.certresolver":          "seelfresolver",
					}, service.Labels)
				} else {
					testutil.DeepEquals(t, types.Labels{
						docker.AppLabel:         string(app.ID()),
						docker.EnvironmentLabel: string(domain.Production),
						"traefik.enable":        "true",
						"traefik.http.services.my-app-production-app.loadbalancer.server.port": "8080",
						"traefik.http.routers.my-app-production-app.rule":                      "Host(`my-app.docker.localhost`)",
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
				if target.Domain().UseSSL() {
					testutil.DeepEquals(t, types.Labels{
						docker.AppLabel:         string(app.ID()),
						docker.EnvironmentLabel: string(domain.Production),
						"traefik.enable":        "true",
						"traefik.http.services.my-app-production-sidecar.loadbalancer.server.port": "80",

						"traefik.http.routers.my-app-production-sidecar.rule":             "Host(`sidecar.my-app.docker.localhost`)",
						"traefik.http.routers.my-app-production-sidecar.tls.certresolver": "seelfresolver",
					}, service.Labels)
				} else {
					testutil.DeepEquals(t, types.Labels{
						docker.AppLabel:         string(app.ID()),
						docker.EnvironmentLabel: string(domain.Production),
						"traefik.enable":        "true",
						"traefik.http.services.my-app-production-sidecar.loadbalancer.server.port": "80",
						"traefik.http.routers.my-app-production-sidecar.rule":                      "Host(`sidecar.my-app.docker.localhost`)",
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
		opts := config.Default(config.WithTestDefaults())
		testServices(t, opts, targetWithoutSSL)
	})

	t.Run("should correctly expose services from a compose file with SSL", func(t *testing.T) {
		opts := config.Default(config.WithTestDefaults())
		testServices(t, opts, targetWithSSL)
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

func (d *dockerCliMockService) Close() error { return nil }

func (d *dockerCliMockService) ImagesPrune(ctx context.Context, pruneFilter filters.Args) (dockertypes.ImagesPruneReport, error) {
	d.pruneFilter = pruneFilter
	return dockertypes.ImagesPruneReport{}, nil
}
