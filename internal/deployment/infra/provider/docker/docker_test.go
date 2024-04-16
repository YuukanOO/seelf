package docker_test

import (
	"context"
	"fmt"
	"os"
	"strings"
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
	"github.com/compose-spec/compose-go/v2/types"
	"github.com/docker/cli/cli/command"
	"github.com/docker/compose/v2/pkg/api"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
)

type options interface {
	artifact.LocalOptions
}

func Test_Docker(t *testing.T) {
	logger := must.Panic(log.NewLogger())
	myAppEnvConfig := domain.NewEnvironmentConfigRequirement(domain.NewEnvironmentConfig("1"), true, true)
	myApp := must.Panic(domain.NewApp("my-app", myAppEnvConfig, myAppEnvConfig, "uid"))
	myAppDeployment := must.Panic(myApp.NewDeployment(1, raw.Data(""), domain.Production, "uid"))

	// Configure specific targets for tests
	targetWithSSL := must.Panic(domain.NewTarget("my-target",
		domain.NewTargetUrlRequirement(must.Panic(domain.UrlFrom("https://docker.localhost")), true),
		domain.NewProviderConfigRequirement(docker.Data{}, true), "uid"))
	evt := testutil.EventIs[domain.TargetCreated](t, &targetWithSSL, 0)
	targetWithSSL.Configured(evt.State.Version(), nil)

	targetWithoutSSL := must.Panic(domain.NewTarget("my-target",
		domain.NewTargetUrlRequirement(must.Panic(domain.UrlFrom("http://docker.localhost")), true),
		domain.NewProviderConfigRequirement(docker.Data{}, true), "uid"))
	evt = testutil.EventIs[domain.TargetCreated](t, &targetWithoutSSL, 0)
	targetWithoutSSL.Configured(evt.State.Version(), nil)

	sut := func(opts options) (domain.Provider, domain.ArtifactManager, *composeMockService, *dockerCliMockService) {
		artifactManager := artifact.NewLocal(opts, logger)
		composeMock := &composeMockService{}
		dockerMock := newDockerMockService()

		t.Cleanup(func() {
			os.RemoveAll(opts.DataDir())
		})

		return docker.New(logger, docker.WithDockerAndCompose(dockerMock, composeMock)), artifactManager, composeMock, dockerMock.cli
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

	t.Run("should be able to configure a target without ssl", func(t *testing.T) {
		opts := config.Default(config.WithTestDefaults())
		provider, _, compose, _ := sut(opts)

		err := provider.Setup(context.Background(), targetWithoutSSL)

		testutil.IsNil(t, err)

		project := compose.up.project

		networkName := fmt.Sprintf("seelf-gateway-%s", strings.ToLower(string(targetWithoutSSL.ID())))
		projectName := fmt.Sprintf("seelf-internal-%s", strings.ToLower(string(targetWithoutSSL.ID())))

		testutil.IsNotNil(t, project)
		testutil.Equals(t, projectName, project.Name)
		testutil.Equals(t, len(project.Services), 1)
		testutil.Equals(t, "proxy", project.Services["proxy"].Name)
		testutil.Equals(t, types.RestartPolicyUnlessStopped, project.Services["proxy"].Restart)
		testutil.Equals(t, "traefik:v2.11", project.Services["proxy"].Image)
		testutil.DeepEquals(t, []string{
			"--providers.docker",
			fmt.Sprintf("--providers.docker.network=%s", networkName),
			fmt.Sprintf("--providers.docker.constraints=Label(`%s`, `%s`) || Label(`%s`, `true`)", docker.TargetLabel, targetWithoutSSL.ID(), docker.ExposedLabel),
			fmt.Sprintf("--providers.docker.defaultrule=Host(`{{ index .Labels %s}}.docker.localhost`)", fmt.Sprintf(`"%s"`, docker.SubdomainLabel)),
		}, project.Services["proxy"].Command)
		testutil.HasLength(t, project.Services["proxy"].Ports, 1)
		testutil.Equals(t, "80", project.Services["proxy"].Ports[0].Published)
		testutil.Equals(t, 80, project.Services["proxy"].Ports[0].Target)
		testutil.HasLength(t, project.Services["proxy"].Volumes, 1)
		testutil.Equals(t, "/var/run/docker.sock", project.Services["proxy"].Volumes[0].Source)
		testutil.Equals(t, "/var/run/docker.sock", project.Services["proxy"].Volumes[0].Target)

		testutil.Equals(t, 1, len(project.Networks))
		testutil.Equals(t, networkName, project.Networks["default"].Name)
	})

	t.Run("should be able to configure a target with ssl", func(t *testing.T) {
		opts := config.Default(config.WithTestDefaults())
		provider, _, compose, _ := sut(opts)

		err := provider.Setup(context.Background(), targetWithSSL)

		testutil.IsNil(t, err)

		project := compose.up.project

		networkName := fmt.Sprintf("seelf-gateway-%s", strings.ToLower(string(targetWithSSL.ID())))
		projectName := fmt.Sprintf("seelf-internal-%s", strings.ToLower(string(targetWithSSL.ID())))
		certResolverName := fmt.Sprintf("seelf-resolver-%s", strings.ToLower(string(targetWithSSL.ID())))

		testutil.IsNotNil(t, project)
		testutil.Equals(t, projectName, project.Name)
		testutil.Equals(t, len(project.Services), 1)
		testutil.Equals(t, "proxy", project.Services["proxy"].Name)
		testutil.Equals(t, types.RestartPolicyUnlessStopped, project.Services["proxy"].Restart)
		testutil.Equals(t, "traefik:v2.11", project.Services["proxy"].Image)
		testutil.DeepEquals(t, []string{
			"--providers.docker",
			fmt.Sprintf("--providers.docker.network=%s", networkName),
			fmt.Sprintf("--providers.docker.constraints=Label(`%s`, `%s`) || Label(`%s`, `true`)", docker.TargetLabel, targetWithSSL.ID(), docker.ExposedLabel),
			fmt.Sprintf("--providers.docker.defaultrule=Host(`{{ index .Labels %s}}.docker.localhost`)", fmt.Sprintf(`"%s"`, docker.SubdomainLabel)),
			"--entrypoints.web.address=:80",
			"--entrypoints.web.http.redirections.entryPoint.to=websecure",
			"--entrypoints.web.http.redirections.entryPoint.scheme=https",
			"--entrypoints.websecure.address=:443",
			fmt.Sprintf("--certificatesresolvers.%s.acme.tlschallenge=true", certResolverName),
			fmt.Sprintf("--certificatesresolvers.%s.acme.storage=/letsencrypt/acme.json", certResolverName),
			fmt.Sprintf("--entrypoints.websecure.http.tls.certresolver=%s", certResolverName),
		}, project.Services["proxy"].Command)
		testutil.HasLength(t, project.Services["proxy"].Ports, 2)
		testutil.Equals(t, "80", project.Services["proxy"].Ports[0].Published)
		testutil.Equals(t, 80, project.Services["proxy"].Ports[0].Target)
		testutil.Equals(t, "443", project.Services["proxy"].Ports[1].Published)
		testutil.Equals(t, 443, project.Services["proxy"].Ports[1].Target)
		testutil.HasLength(t, project.Services["proxy"].Volumes, 2)
		testutil.Equals(t, "/var/run/docker.sock", project.Services["proxy"].Volumes[0].Source)
		testutil.Equals(t, "/var/run/docker.sock", project.Services["proxy"].Volumes[0].Target)
		testutil.Equals(t, "letsencrypt", project.Services["proxy"].Volumes[1].Source)
		testutil.Equals(t, "/letsencrypt", project.Services["proxy"].Volumes[1].Target)

		testutil.Equals(t, 1, len(project.Networks))
		testutil.Equals(t, networkName, project.Networks["default"].Name)

		testutil.Equals(t, 1, len(project.Volumes))
		testutil.Equals(t, fmt.Sprintf("%s_letsencrypt", projectName), project.Volumes["letsencrypt"].Name)
	})

	t.Run("should be able to cleanup a target", func(t *testing.T) {
		provider, _, _, _ := sut(config.Default(config.WithTestDefaults()))

		err := provider.CleanupTarget(context.Background(), targetWithSSL, domain.CleanupStrategyDefault)

		testutil.IsNil(t, err)
	})

	t.Run("should be able to cleanup an app environment on a target", func(t *testing.T) {
		provider, _, _, _ := sut(config.Default(config.WithTestDefaults()))

		err := provider.Cleanup(context.Background(), myApp.ID(), targetWithSSL, myAppDeployment.Config().Environment(), domain.CleanupStrategyDefault)

		testutil.IsNil(t, err)
	})

	t.Run("should err if no compose file was found for a deployment", func(t *testing.T) {
		provider, artifactManager, _, _ := sut(config.Default(config.WithTestDefaults()))
		ctx := context.Background()
		deplCtx := must.Panic(artifactManager.PrepareBuild(ctx, myAppDeployment))
		defer deplCtx.Logger().Close()

		_, err := provider.Deploy(ctx, deplCtx, myAppDeployment, targetWithoutSSL)

		testutil.ErrorIs(t, docker.ErrOpenComposeFileFailed, err)
	})

	// Main function which asserts for a successful deployment
	assertDeployedProject := func(t *testing.T, opts options, target domain.Target) {
		provider, artifactManager, composeMock, cliMock := sut(opts)

		dsn := "postgres://prodapp:passprod@db/app?sslmode=disable"
		postgresUser := "prodapp"
		postgresPassword := "passprod"
		targetNetworkName := fmt.Sprintf("seelf-gateway-%s", strings.ToLower(string(target.ID())))

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
			domain.NewEnvironmentConfigRequirement(config, true, true),
			domain.NewEnvironmentConfigRequirement(domain.NewEnvironmentConfig(target.ID()), true, true),
			"uid",
		))
		appidLower := strings.ToLower(string(app.ID()))
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
		deplCtx := must.Panic(artifactManager.PrepareBuild(ctx, depl))

		defer deplCtx.Logger().Close()

		testutil.IsNil(t, src.Fetch(ctx, deplCtx, depl))

		services, err := provider.Deploy(ctx, deplCtx, depl, target)

		testutil.IsNil(t, err)
		testutil.HasLength(t, services, 3)
		testutil.Equals(t, "app", services[0].Name())
		testutil.Equals(t, fmt.Sprintf("my-app-%s/app:production", appidLower), services[0].Image())
		testutil.Equals(t, "my-app", services[0].Subdomain().MustGet())

		testutil.Equals(t, "db", services[1].Name())
		testutil.Equals(t, "postgres:14-alpine", services[1].Image())
		testutil.IsFalse(t, services[1].Subdomain().HasValue())

		testutil.Equals(t, "sidecar", services[2].Name())
		testutil.Equals(t, "traefik/whoami", services[2].Image())
		testutil.Equals(t, "sidecar.my-app", services[2].Subdomain().MustGet())

		project := composeMock.up.project
		testutil.IsNotNil(t, project)
		testutil.Equals(t, fmt.Sprintf("my-app-production-%s", appidLower), project.Name)

		for _, service := range project.Services {
			switch service.Name {
			case "app":
				testutil.Equals(t, fmt.Sprintf("my-app-%s/app:production", appidLower), service.Image)
				testutil.Equals(t, types.RestartPolicyUnlessStopped, service.Restart)
				testutil.Equals(t, types.PullPolicyBuild, service.PullPolicy)
				testutil.DeepEquals(t, types.Labels{
					docker.AppLabel:         string(app.ID()),
					docker.TargetLabel:      string(target.ID()),
					docker.EnvironmentLabel: string(domain.Production),
				}, service.Build.Labels)
				testutil.HasLength(t, service.Ports, 0)
				testutil.DeepEquals(t, types.MappingWithEquals{
					"DSN": &dsn,
				}, service.Environment)
				testutil.DeepEquals(t, types.Labels{
					docker.AppLabel:         string(app.ID()),
					docker.TargetLabel:      string(target.ID()),
					docker.EnvironmentLabel: string(domain.Production),
					docker.SubdomainLabel:   "my-app",
					fmt.Sprintf("traefik.http.services.my-app-production-%s-app.loadbalancer.server.port", appidLower): "8080",
				}, service.Labels)
			case "db":
				testutil.Equals(t, "postgres:14-alpine", service.Image)
				testutil.Equals(t, types.RestartPolicyUnlessStopped, service.Restart)
				testutil.DeepEquals(t, types.Labels{
					docker.AppLabel:         string(app.ID()),
					docker.TargetLabel:      string(target.ID()),
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
				testutil.DeepEquals(t, types.Labels{
					docker.AppLabel:         string(app.ID()),
					docker.TargetLabel:      string(target.ID()),
					docker.SubdomainLabel:   "sidecar.my-app",
					docker.EnvironmentLabel: string(domain.Production),
					fmt.Sprintf("traefik.http.services.my-app-production-%s-sidecar.loadbalancer.server.port", appidLower): "80",
				}, service.Labels)
			default:
				t.Fatal("no other service expected")
			}
		}

		testutil.Equals(t, 2, len(project.Networks))
		testutil.Equals(t, fmt.Sprintf("my-app-production-%s_default", appidLower), project.Networks["default"].Name)
		testutil.DeepEquals(t, types.Labels{
			docker.AppLabel:         string(app.ID()),
			docker.TargetLabel:      string(target.ID()),
			docker.EnvironmentLabel: string(domain.Production),
		}, project.Networks["default"].Labels)
		testutil.Equals(t, targetNetworkName, project.Networks[targetNetworkName].Name)
		testutil.Equals(t, 0, len(project.Networks[targetNetworkName].Labels))
		testutil.IsTrue(t, project.Networks[targetNetworkName].External)

		testutil.Equals(t, 1, len(project.Volumes))
		testutil.Equals(t, fmt.Sprintf("my-app-production-%s_dbdata", appidLower), project.Volumes["dbdata"].Name)
		testutil.DeepEquals(t, types.Labels{
			docker.AppLabel:         string(app.ID()),
			docker.TargetLabel:      string(target.ID()),
			docker.EnvironmentLabel: string(domain.Production),
		}, project.Volumes["dbdata"].Labels)

		testutil.DeepEquals(t, filters.NewArgs(
			filters.Arg("dangling", "true"),
			filters.Arg("label", fmt.Sprintf("%s=%s", docker.AppLabel, app.ID())),
			filters.Arg("label", fmt.Sprintf("%s=%s", docker.TargetLabel, target.ID())),
			filters.Arg("label", fmt.Sprintf("%s=%s", docker.EnvironmentLabel, domain.Production)),
		), cliMock.pruneFilter)
	}

	t.Run("should correctly expose services from a compose file without SSL", func(t *testing.T) {
		opts := config.Default(config.WithTestDefaults())
		assertDeployedProject(t, opts, targetWithoutSSL)
	})

	t.Run("should correctly expose services from a compose file with SSL", func(t *testing.T) {
		opts := config.Default(config.WithTestDefaults())
		assertDeployedProject(t, opts, targetWithSSL)
	})
}

type composeMockService struct {
	api.Service
	up struct {
		project *types.Project
		options api.UpOptions
	}
}

func (c *composeMockService) Up(ctx context.Context, project *types.Project, options api.UpOptions) error {
	c.up.project = project
	c.up.options = options
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

func (d *dockerMockService) Apply(ops ...command.CLIOption) error {
	return nil
}

func (d *dockerCliMockService) Close() error { return nil }

func (d *dockerCliMockService) ImagesPrune(_ context.Context, pruneFilter filters.Args) (dockertypes.ImagesPruneReport, error) {
	d.pruneFilter = pruneFilter
	return dockertypes.ImagesPruneReport{}, nil
}

func (d *dockerCliMockService) ContainerList(context.Context, container.ListOptions) ([]dockertypes.Container, error) {
	return nil, nil
}

func (d *dockerCliMockService) VolumeList(context.Context, volume.ListOptions) (volume.ListResponse, error) {
	return volume.ListResponse{}, nil
}

func (d *dockerCliMockService) NetworkList(context.Context, dockertypes.NetworkListOptions) ([]dockertypes.NetworkResource, error) {
	return nil, nil
}

func (d *dockerCliMockService) ImageList(context.Context, dockertypes.ImageListOptions) ([]image.Summary, error) {
	return nil, nil
}
