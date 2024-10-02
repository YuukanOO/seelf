package docker_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/YuukanOO/seelf/cmd/config"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/deployment/fixture"
	"github.com/YuukanOO/seelf/internal/deployment/infra/artifact"
	"github.com/YuukanOO/seelf/internal/deployment/infra/provider/docker"
	"github.com/YuukanOO/seelf/internal/deployment/infra/source/raw"
	"github.com/YuukanOO/seelf/pkg/assert"
	"github.com/YuukanOO/seelf/pkg/log"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/must"
	"github.com/YuukanOO/seelf/pkg/ssh"
	"github.com/compose-spec/compose-go/v2/types"
	"github.com/docker/cli/cli/command"
	"github.com/docker/compose/v2/pkg/api"
	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

type options interface {
	artifact.LocalOptions
}

func Test_Provider(t *testing.T) {
	logger := must.Panic(log.NewLogger())

	arrange := func(opts options) (docker.Docker, *dockerMockService) {
		mock := newMockService()

		t.Cleanup(func() {
			os.RemoveAll(opts.DataDir())
		})

		return docker.New(logger, docker.WithTestConfig(mock, mock, filepath.Join(opts.DataDir(), "config"))), mock
	}

	t.Run("should be able to prepare a docker provider config from a raw payload", func(t *testing.T) {
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
				existing: []domain.ProviderConfig{
					docker.Data{
						Host:       monad.Value[ssh.Host]("other"),
						PrivateKey: monad.Value[ssh.PrivateKey]("other"),
					},
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
				},
				existing: []domain.ProviderConfig{
					docker.Data{
						Host:       monad.Value[ssh.Host]("other"),
						PrivateKey: monad.Value[ssh.PrivateKey]("other"),
					},
				},
				expected: docker.Data{
					Host:       monad.Value[ssh.Host]("localhost"),
					User:       monad.Value("test"),
					Port:       monad.Value(2222),
					PrivateKey: monad.Value[ssh.PrivateKey]("other"),
				},
			},
			{
				payload: docker.Body{
					Host:       monad.Value("localhost"),
					PrivateKey: monad.Nil[string](),
				},
				existing: []domain.ProviderConfig{
					docker.Data{
						Host:       monad.Value[ssh.Host]("other"),
						PrivateKey: monad.Value[ssh.PrivateKey]("other"),
					},
				},
				expected: docker.Data{
					Host: monad.Value[ssh.Host]("localhost"),
					User: monad.Value("docker"),
					Port: monad.Value(22),
				},
			},
		}

		provider, _ := arrange(config.Default(config.WithTestDefaults()))

		for _, tt := range tests {
			t.Run(fmt.Sprintf("%v", tt.payload), func(t *testing.T) {
				data, err := provider.Prepare(context.Background(), tt.payload, tt.existing...)

				assert.Nil(t, err)
				assert.True(t, data.Equals(tt.expected))
			})
		}
	})

	t.Run("should correctly setup needed stuff on a target", func(t *testing.T) {
		t.Run("with automatic proxy, no-ssl, no custom entrypoints", func(t *testing.T) {
			target := fixture.Target(fixture.WithProviderConfig(docker.Data{}))
			assert.Nil(t, target.ExposeServicesAutomatically(domain.NewTargetUrlRequirement(must.Panic(domain.UrlFrom("http://docker.localhost")), true)))
			targetIdLower := strings.ToLower(string(target.ID()))
			provider, mock := arrange(config.Default(config.WithTestDefaults()))

			assigned, err := provider.Setup(context.Background(), target)

			assert.Nil(t, err)
			assert.DeepEqual(t, domain.TargetEntrypointsAssigned{}, assigned)
			assert.HasLength(t, 1, mock.ups)
			assert.DeepEqual(t, &types.Project{
				Name: "seelf-internal-" + targetIdLower,
				Services: types.Services{
					"proxy": {
						Name: "proxy",
						Labels: types.Labels{
							docker.TargetLabel: string(target.ID()),
						},
						Image:   "traefik:v2.11",
						Restart: types.RestartPolicyUnlessStopped,
						Command: types.ShellCommand{
							"--entrypoints.http.address=:80",
							"--providers.docker",
							fmt.Sprintf("--providers.docker.constraints=(Label(`%s`, `%s`) && (Label(`%s`, `true`) || LabelRegex(`%s`, `.+`))) || Label(`%s`, `true`)",
								docker.TargetLabel, target.ID(), docker.CustomEntrypointsLabel, docker.SubdomainLabel, docker.ExposedLabel),
							fmt.Sprintf("--providers.docker.defaultrule=Host(`{{ index .Labels %s}}.docker.localhost`)", fmt.Sprintf(`"%s"`, docker.SubdomainLabel)),
							"--providers.docker.network=seelf-gateway-" + targetIdLower,
						},
						Ports: []types.ServicePortConfig{
							{Target: 80, Published: "80"},
						},
						Volumes: []types.ServiceVolumeConfig{
							{Type: types.VolumeTypeBind, Source: "/var/run/docker.sock", Target: "/var/run/docker.sock"},
						},
						CustomLabels: types.Labels{
							api.ProjectLabel:     "seelf-internal-" + targetIdLower,
							api.ServiceLabel:     "proxy",
							api.VersionLabel:     api.ComposeVersion,
							api.ConfigFilesLabel: "",
							api.OneoffLabel:      "False",
						},
					},
				},
				Networks: types.Networks{
					"default": types.NetworkConfig{
						Name: "seelf-gateway-" + targetIdLower,
						Labels: types.Labels{
							docker.TargetLabel: string(target.ID()),
						},
					},
				},
			}, mock.ups[0].project)
		})

		t.Run("with automatic proxy, ssl, no custom entrypoints", func(t *testing.T) {
			target := fixture.Target(fixture.WithProviderConfig(docker.Data{}))
			assert.Nil(t, target.ExposeServicesAutomatically(domain.NewTargetUrlRequirement(must.Panic(domain.UrlFrom("https://docker.localhost")), true)))
			targetIdLower := strings.ToLower(string(target.ID()))
			provider, mock := arrange(config.Default(config.WithTestDefaults()))

			assigned, err := provider.Setup(context.Background(), target)

			assert.Nil(t, err)
			assert.DeepEqual(t, domain.TargetEntrypointsAssigned{}, assigned)
			assert.HasLength(t, 1, mock.ups)
			assert.DeepEqual(t, &types.Project{
				Name: "seelf-internal-" + targetIdLower,
				Services: types.Services{
					"proxy": {
						Name: "proxy",
						Labels: types.Labels{
							docker.TargetLabel: string(target.ID()),
						},
						Image:   "traefik:v2.11",
						Restart: types.RestartPolicyUnlessStopped,
						Command: types.ShellCommand{
							fmt.Sprintf("--certificatesresolvers.%s.acme.storage=/letsencrypt/acme.json", "seelf-resolver-"+targetIdLower),
							fmt.Sprintf("--certificatesresolvers.%s.acme.tlschallenge=true", "seelf-resolver-"+targetIdLower),
							"--entrypoints.http.address=:443",
							"--entrypoints.http.http.tls.certresolver=seelf-resolver-" + targetIdLower,
							"--entrypoints.insecure.address=:80",
							"--entrypoints.insecure.http.redirections.entryPoint.scheme=https",
							"--entrypoints.insecure.http.redirections.entryPoint.to=http",
							"--providers.docker",
							fmt.Sprintf("--providers.docker.constraints=(Label(`%s`, `%s`) && (Label(`%s`, `true`) || LabelRegex(`%s`, `.+`))) || Label(`%s`, `true`)",
								docker.TargetLabel, target.ID(), docker.CustomEntrypointsLabel, docker.SubdomainLabel, docker.ExposedLabel),
							fmt.Sprintf("--providers.docker.defaultrule=Host(`{{ index .Labels %s}}.docker.localhost`)", fmt.Sprintf(`"%s"`, docker.SubdomainLabel)),
							"--providers.docker.network=seelf-gateway-" + targetIdLower,
						},
						Ports: []types.ServicePortConfig{
							{Target: 80, Published: "80"},
							{Target: 443, Published: "443"},
						},
						Volumes: []types.ServiceVolumeConfig{
							{Type: types.VolumeTypeBind, Source: "/var/run/docker.sock", Target: "/var/run/docker.sock"},
							{Type: types.VolumeTypeVolume, Source: "letsencrypt", Target: "/letsencrypt"},
						},
						CustomLabels: types.Labels{
							api.ProjectLabel:     "seelf-internal-" + targetIdLower,
							api.ServiceLabel:     "proxy",
							api.VersionLabel:     api.ComposeVersion,
							api.ConfigFilesLabel: "",
							api.OneoffLabel:      "False",
						},
					},
				},
				Networks: types.Networks{
					"default": types.NetworkConfig{
						Name: "seelf-gateway-" + targetIdLower,
						Labels: types.Labels{
							docker.TargetLabel: string(target.ID()),
						},
					},
				},
				Volumes: types.Volumes{
					"letsencrypt": types.VolumeConfig{
						Name: "seelf-internal-" + targetIdLower + "_letsencrypt",
						Labels: types.Labels{
							docker.TargetLabel: string(target.ID()),
						},
					},
				},
			}, mock.ups[0].project)
		})

		t.Run("with automatic proxy, custom entrypoints", func(t *testing.T) {
			target := fixture.Target(fixture.WithProviderConfig(docker.Data{}))
			assert.Nil(t, target.ExposeServicesAutomatically(domain.NewTargetUrlRequirement(must.Panic(domain.UrlFrom("http://docker.localhost")), true)))
			targetIdLower := strings.ToLower(string(target.ID()))
			app := fixture.App(fixture.WithAppName("my-app"), fixture.WithEnvironmentConfig(
				domain.NewEnvironmentConfig(target.ID()),
				domain.NewEnvironmentConfig(target.ID()),
			))
			deployment := fixture.Deployment(fixture.FromApp(app))
			builder := deployment.Config().ServicesBuilder()
			service := builder.AddService("app", "")
			tcp := service.AddTCPEntrypoint(5432, true)
			udp := service.AddUDPEntrypoint(5433, true)
			target.ExposeEntrypoints(deployment.Config().AppID(), deployment.Config().Environment(), builder.Services())
			provider, mock := arrange(config.Default(config.WithTestDefaults()))

			assigned, err := provider.Setup(context.Background(), target)

			assert.Nil(t, err)
			assert.HasLength(t, 2, mock.ups, "should have launch two projects since it has to find available ports")
			assert.HasLength(t, 1, mock.downs)
			tcpPort := assigned[deployment.ID().AppID()][deployment.Config().Environment()][tcp.Name()]
			udpPort := assigned[deployment.ID().AppID()][deployment.Config().Environment()][udp.Name()]

			assert.NotZero(t, tcpPort)
			assert.NotZero(t, udpPort)

			assert.DeepEqual(t, &types.Project{
				Name: "seelf-internal-" + targetIdLower,
				Services: types.Services{
					"proxy": {
						Name: "proxy",
						Labels: types.Labels{
							docker.TargetLabel: string(target.ID()),
						},
						Image:   "traefik:v2.11",
						Restart: types.RestartPolicyUnlessStopped,
						Command: types.ShellCommand{
							"--entrypoints.http.address=:80",
							fmt.Sprintf("--entrypoints.%s.address=:%d/tcp", tcp.Name(), tcpPort),
							fmt.Sprintf("--entrypoints.%s.address=:%d/udp", udp.Name(), udpPort),
							"--providers.docker",
							fmt.Sprintf("--providers.docker.constraints=(Label(`%s`, `%s`) && (Label(`%s`, `true`) || LabelRegex(`%s`, `.+`))) || Label(`%s`, `true`)",
								docker.TargetLabel, target.ID(), docker.CustomEntrypointsLabel, docker.SubdomainLabel, docker.ExposedLabel),
							fmt.Sprintf("--providers.docker.defaultrule=Host(`{{ index .Labels %s}}.docker.localhost`)", fmt.Sprintf(`"%s"`, docker.SubdomainLabel)),
							"--providers.docker.network=seelf-gateway-" + targetIdLower,
						},
						Ports: sortedPorts([]types.ServicePortConfig{
							{Target: 80, Published: "80"},
							{Target: uint32(tcpPort), Published: strconv.FormatUint(uint64(tcpPort), 10), Protocol: "tcp"},
							{Target: uint32(udpPort), Published: strconv.FormatUint(uint64(udpPort), 10), Protocol: "udp"},
						}),
						Volumes: []types.ServiceVolumeConfig{
							{Type: types.VolumeTypeBind, Source: "/var/run/docker.sock", Target: "/var/run/docker.sock"},
						},
						CustomLabels: types.Labels{
							api.ProjectLabel:     "seelf-internal-" + targetIdLower,
							api.ServiceLabel:     "proxy",
							api.VersionLabel:     api.ComposeVersion,
							api.ConfigFilesLabel: "",
							api.OneoffLabel:      "False",
						},
					},
				},
				Networks: types.Networks{
					"default": types.NetworkConfig{
						Name: "seelf-gateway-" + targetIdLower,
						Labels: types.Labels{
							docker.TargetLabel: string(target.ID()),
						},
					},
				},
			}, mock.ups[1].project)
		})

		t.Run("with automatic proxy, custom entrypoints and already determined ports", func(t *testing.T) {
			target := fixture.Target(fixture.WithProviderConfig(docker.Data{}))
			assert.Nil(t, target.ExposeServicesAutomatically(domain.NewTargetUrlRequirement(must.Panic(domain.UrlFrom("http://docker.localhost")), true)))
			targetIdLower := strings.ToLower(string(target.ID()))
			app := fixture.App(fixture.WithAppName("my-app"), fixture.WithEnvironmentConfig(
				domain.NewEnvironmentConfig(target.ID()),
				domain.NewEnvironmentConfig(target.ID()),
			))
			deployment := fixture.Deployment(fixture.FromApp(app))
			builder := deployment.Config().ServicesBuilder()
			service := builder.AddService("app", "")
			tcp := service.AddTCPEntrypoint(5432, true)
			udp := service.AddUDPEntrypoint(5433, true)
			target.ExposeEntrypoints(deployment.ID().AppID(), deployment.Config().Environment(), builder.Services())
			assert.Nil(t, target.Configured(target.CurrentVersion(), domain.TargetEntrypointsAssigned{
				deployment.ID().AppID(): {
					deployment.Config().Environment(): {
						tcp.Name(): 5432,
						udp.Name(): 5433,
					},
				},
			}, nil))
			newTcp := service.AddTCPEntrypoint(5434, true)
			newUdp := service.AddUDPEntrypoint(5435, true)
			target.ExposeEntrypoints(deployment.ID().AppID(), deployment.Config().Environment(), builder.Services())
			provider, mock := arrange(config.Default(config.WithTestDefaults()))

			assigned, err := provider.Setup(context.Background(), target)

			assert.Nil(t, err)
			assert.HasLength(t, 2, mock.ups)
			assert.HasLength(t, 1, mock.downs)
			assert.Equal(t, 2, len(assigned[deployment.ID().AppID()][deployment.Config().Environment()]))

			tcpPort := assigned[deployment.ID().AppID()][deployment.Config().Environment()][newTcp.Name()]
			udpPort := assigned[deployment.ID().AppID()][deployment.Config().Environment()][newUdp.Name()]

			assert.NotZero(t, tcpPort)
			assert.NotZero(t, udpPort)

			assert.DeepEqual(t, &types.Project{
				Name: "seelf-internal-" + targetIdLower,
				Services: types.Services{
					"proxy": {
						Name: "proxy",
						Labels: types.Labels{
							docker.TargetLabel: string(target.ID()),
						},
						Image:   "traefik:v2.11",
						Restart: types.RestartPolicyUnlessStopped,
						Command: types.ShellCommand{
							"--entrypoints.http.address=:80",
							fmt.Sprintf("--entrypoints.%s.address=:5432/tcp", tcp.Name()),
							fmt.Sprintf("--entrypoints.%s.address=:5433/udp", udp.Name()),
							fmt.Sprintf("--entrypoints.%s.address=:%d/tcp", newTcp.Name(), tcpPort),
							fmt.Sprintf("--entrypoints.%s.address=:%d/udp", newUdp.Name(), udpPort),
							"--providers.docker",
							fmt.Sprintf("--providers.docker.constraints=(Label(`%s`, `%s`) && (Label(`%s`, `true`) || LabelRegex(`%s`, `.+`))) || Label(`%s`, `true`)",
								docker.TargetLabel, target.ID(), docker.CustomEntrypointsLabel, docker.SubdomainLabel, docker.ExposedLabel),
							fmt.Sprintf("--providers.docker.defaultrule=Host(`{{ index .Labels %s}}.docker.localhost`)", fmt.Sprintf(`"%s"`, docker.SubdomainLabel)),
							"--providers.docker.network=seelf-gateway-" + targetIdLower,
						},
						Ports: sortedPorts([]types.ServicePortConfig{
							{Target: 80, Published: "80"},
							{Target: 5432, Published: "5432", Protocol: "tcp"},
							{Target: 5433, Published: "5433", Protocol: "udp"},
							{Target: uint32(tcpPort), Published: strconv.FormatUint(uint64(tcpPort), 10), Protocol: "tcp"},
							{Target: uint32(udpPort), Published: strconv.FormatUint(uint64(udpPort), 10), Protocol: "udp"},
						}),
						Volumes: []types.ServiceVolumeConfig{
							{Type: types.VolumeTypeBind, Source: "/var/run/docker.sock", Target: "/var/run/docker.sock"},
						},
						CustomLabels: types.Labels{
							api.ProjectLabel:     "seelf-internal-" + targetIdLower,
							api.ServiceLabel:     "proxy",
							api.VersionLabel:     api.ComposeVersion,
							api.ConfigFilesLabel: "",
							api.OneoffLabel:      "False",
						},
					},
				},
				Networks: types.Networks{
					"default": types.NetworkConfig{
						Name: "seelf-gateway-" + targetIdLower,
						Labels: types.Labels{
							docker.TargetLabel: string(target.ID()),
						},
					},
				},
			}, mock.ups[1].project)
		})

		t.Run("with manual target, should not deploy the proxy and remove existing one", func(t *testing.T) {
			target := fixture.Target(fixture.WithProviderConfig(docker.Data{}))
			provider, mock := arrange(config.Default(config.WithTestDefaults()))

			assigned, err := provider.Setup(context.Background(), target)

			assert.Nil(t, err)
			assert.True(t, assigned == nil)
			assert.HasLength(t, 0, mock.ups)
			assert.HasLength(t, 1, mock.downs)
			assert.Equal(t, "seelf-internal-"+strings.ToLower(string(target.ID())), mock.downs[0].projectName)
		})
	})

	t.Run("should be able to process a deployment", func(t *testing.T) {
		t.Run("should returns an error if no valid compose file was found", func(t *testing.T) {
			target := fixture.Target(fixture.WithProviderConfig(docker.Data{}))
			deployment := fixture.Deployment()
			opts := config.Default(config.WithTestDefaults())
			artifactManager := artifact.NewLocal(opts, logger)
			ctx, err := artifactManager.PrepareBuild(context.Background(), deployment)
			assert.Nil(t, err)
			defer ctx.Logger().Close()
			provider, _ := arrange(opts)

			_, err = provider.Deploy(context.Background(), ctx, deployment, target, nil)

			assert.ErrorIs(t, docker.ErrOpenComposeFileFailed, err)
		})

		t.Run("should correctly transform the compose file if the target is configured as automatically exposing services", func(t *testing.T) {
			target := fixture.Target(fixture.WithProviderConfig(docker.Data{}))
			assert.Nil(t, target.ExposeServicesAutomatically(domain.NewTargetUrlRequirement(must.Panic(domain.UrlFrom("http://docker.localhost")), true)))
			productionConfig := domain.NewEnvironmentConfig(target.ID())
			productionConfig.HasEnvironmentVariables(domain.ServicesEnv{
				"app": domain.EnvVars{
					"DSN": "postgres://prodapp:passprod@db/app?sslmode=disable",
				},
				"db": domain.EnvVars{
					"POSTGRES_USER":     "prodapp",
					"POSTGRES_PASSWORD": "passprod",
				},
			})
			app := fixture.App(
				fixture.WithAppName("my-app"),
				fixture.WithEnvironmentConfig(
					productionConfig,
					domain.NewEnvironmentConfig(target.ID()),
				),
			)
			deployment := fixture.Deployment(
				fixture.FromApp(app),
				fixture.ForEnvironment(domain.Production),
				fixture.WithSourceData(raw.Data(`services:
  sidecar:
    image: traefik/whoami
    profiles:
      - production
  app:
    restart: unless-stopped
    build: .
    environment:
      - DSN=postgres://app:apppa55word@db/app?sslmode=disable
    depends_on:
      - db
    ports:
      - "8080:8080"
      - "8081:8081/udp"
      - "8082:8082"
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
    ports:
      - "5432:5432/tcp"
volumes:
  dbdata:`)),
			)
			appIdLower := strings.ToLower(string(app.ID()))

			// Prepare the build
			opts := config.Default(config.WithTestDefaults())
			artifactManager := artifact.NewLocal(opts, logger)
			deploymentContext, err := artifactManager.PrepareBuild(context.Background(), deployment)
			assert.Nil(t, err)
			assert.Nil(t, raw.New().Fetch(context.Background(), deploymentContext, deployment))
			defer deploymentContext.Logger().Close()
			provider, mock := arrange(opts)

			services, err := provider.Deploy(context.Background(), deploymentContext, deployment, target, nil)

			assert.Nil(t, err)
			assert.HasLength(t, 1, mock.ups)
			assert.HasLength(t, 3, services)

			assert.Equal(t, "app", services[0].Name())
			assert.Equal(t, "db", services[1].Name())
			assert.Equal(t, "sidecar", services[2].Name())

			entrypoints := services.Entrypoints()
			assert.HasLength(t, 4, entrypoints)
			assert.Equal(t, 8080, entrypoints[0].Port())
			assert.Equal(t, "http", entrypoints[0].Router())
			assert.Equal(t, string(deployment.Config().AppName()), entrypoints[0].Subdomain().Get(""))
			assert.Equal(t, 8081, entrypoints[1].Port())
			assert.Equal(t, "udp", entrypoints[1].Router())
			assert.Equal(t, 8082, entrypoints[2].Port())
			assert.Equal(t, "http", entrypoints[2].Router())
			assert.Equal(t, string(deployment.Config().AppName()), entrypoints[2].Subdomain().Get(""))
			assert.Equal(t, 5432, entrypoints[3].Port())
			assert.Equal(t, "tcp", entrypoints[3].Router())

			project := mock.ups[0].project
			expectedProjectName := fmt.Sprintf("%s-%s-%s", deployment.Config().AppName(), deployment.Config().Environment(), appIdLower)
			expectedGatewayNetworkName := "seelf-gateway-" + strings.ToLower(string(target.ID()))
			assert.Equal(t, expectedProjectName, project.Name)
			assert.Equal(t, 3, len(project.Services))

			for _, service := range project.Services {
				switch service.Name {
				case "sidecar":
					assert.Equal(t, "traefik/whoami", service.Image)
					assert.HasLength(t, 0, service.Ports)
					assert.DeepEqual(t, types.MappingWithEquals{}, service.Environment)
					assert.DeepEqual(t, types.Labels{
						docker.AppLabel:         string(deployment.ID().AppID()),
						docker.TargetLabel:      string(target.ID()),
						docker.EnvironmentLabel: string(deployment.Config().Environment()),
					}, service.Labels)
					assert.DeepEqual(t, map[string]*types.ServiceNetworkConfig{
						"default": nil,
					}, service.Networks)
				case "app":
					httpEntrypointName := string(entrypoints[0].Name())
					udpEntrypointName := string(entrypoints[1].Name())
					customHttpEntrypointName := string(entrypoints[2].Name())
					dsn := deployment.Config().EnvironmentVariablesFor("app").MustGet()["DSN"]

					assert.Equal(t, fmt.Sprintf("%s-%s/app:%s", deployment.Config().AppName(), appIdLower, deployment.Config().Environment()), service.Image)
					assert.Equal(t, types.RestartPolicyUnlessStopped, service.Restart)
					assert.DeepEqual(t, types.Labels{
						docker.AppLabel:         string(deployment.ID().AppID()),
						docker.TargetLabel:      string(target.ID()),
						docker.EnvironmentLabel: string(deployment.Config().Environment()),
						docker.SubdomainLabel:   string(deployment.Config().AppName()),
						fmt.Sprintf("traefik.http.routers.%s.entrypoints", httpEntrypointName):               "http",
						fmt.Sprintf("traefik.http.routers.%s.service", httpEntrypointName):                   httpEntrypointName,
						fmt.Sprintf("traefik.http.services.%s.loadbalancer.server.port", httpEntrypointName): "8080",
						docker.CustomEntrypointsLabel:                                                              "true",
						fmt.Sprintf("traefik.udp.routers.%s.entrypoints", udpEntrypointName):                       udpEntrypointName,
						fmt.Sprintf("traefik.udp.routers.%s.service", udpEntrypointName):                           udpEntrypointName,
						fmt.Sprintf("traefik.udp.services.%s.loadbalancer.server.port", udpEntrypointName):         "8081",
						fmt.Sprintf("traefik.http.routers.%s.entrypoints", customHttpEntrypointName):               customHttpEntrypointName,
						fmt.Sprintf("traefik.http.routers.%s.service", customHttpEntrypointName):                   customHttpEntrypointName,
						fmt.Sprintf("traefik.http.services.%s.loadbalancer.server.port", customHttpEntrypointName): "8082",
					}, service.Labels)

					assert.HasLength(t, 0, service.Ports)
					assert.DeepEqual(t, types.MappingWithEquals{
						"DSN": &dsn,
					}, service.Environment)
					assert.DeepEqual(t, map[string]*types.ServiceNetworkConfig{
						"default":                  nil,
						expectedGatewayNetworkName: nil,
					}, service.Networks)
				case "db":
					entrypointName := string(entrypoints[3].Name())
					postgresUser := deployment.Config().EnvironmentVariablesFor("db").MustGet()["POSTGRES_USER"]
					postgresPassword := deployment.Config().EnvironmentVariablesFor("db").MustGet()["POSTGRES_PASSWORD"]

					assert.Equal(t, "postgres:14-alpine", service.Image)
					assert.Equal(t, types.RestartPolicyUnlessStopped, service.Restart)
					assert.DeepEqual(t, types.Labels{
						docker.AppLabel:         string(deployment.ID().AppID()),
						docker.TargetLabel:      string(target.ID()),
						docker.EnvironmentLabel: string(deployment.Config().Environment()),
						fmt.Sprintf("traefik.tcp.routers.%s.rule", entrypointName):                      "HostSNI(`*`)",
						docker.CustomEntrypointsLabel:                                                   "true",
						fmt.Sprintf("traefik.tcp.routers.%s.entrypoints", entrypointName):               entrypointName,
						fmt.Sprintf("traefik.tcp.routers.%s.service", entrypointName):                   entrypointName,
						fmt.Sprintf("traefik.tcp.services.%s.loadbalancer.server.port", entrypointName): "5432",
					}, service.Labels)
					assert.HasLength(t, 0, service.Ports)
					assert.DeepEqual(t, types.MappingWithEquals{
						"POSTGRES_USER":     &postgresUser,
						"POSTGRES_PASSWORD": &postgresPassword,
					}, service.Environment)
					assert.DeepEqual(t, map[string]*types.ServiceNetworkConfig{
						"default":                  nil,
						expectedGatewayNetworkName: nil,
					}, service.Networks)
					assert.DeepEqual(t, []types.ServiceVolumeConfig{
						{
							Type:   types.VolumeTypeVolume,
							Source: "dbdata",
							Target: "/var/lib/postgresql/data",
							Volume: &types.ServiceVolumeVolume{},
						},
					}, service.Volumes)
				default:
					t.Fatalf("unexpected service %s", service.Name)
				}
			}

			assert.DeepEqual(t, types.Networks{
				"default": {
					Name: expectedProjectName + "_default",
					Labels: types.Labels{
						docker.TargetLabel:      string(target.ID()),
						docker.AppLabel:         string(deployment.Config().AppID()),
						docker.EnvironmentLabel: string(deployment.Config().Environment()),
					},
				},
				expectedGatewayNetworkName: {
					Name:     expectedGatewayNetworkName,
					External: true,
				},
			}, project.Networks)
			assert.DeepEqual(t, types.Volumes{
				"dbdata": {
					Name: expectedProjectName + "_dbdata",
					Labels: types.Labels{
						docker.TargetLabel:      string(target.ID()),
						docker.AppLabel:         string(deployment.Config().AppID()),
						docker.EnvironmentLabel: string(deployment.Config().Environment()),
					},
				},
			}, project.Volumes)

			assert.DeepEqual(t, filters.NewArgs(
				filters.Arg("dangling", "true"),
				filters.Arg("label", fmt.Sprintf("%s=%s", docker.AppLabel, deployment.ID().AppID())),
				filters.Arg("label", fmt.Sprintf("%s=%s", docker.TargetLabel, target.ID())),
				filters.Arg("label", fmt.Sprintf("%s=%s", docker.EnvironmentLabel, deployment.Config().Environment())),
			), mock.pruneFilters)
		})

		t.Run("should correctly transform the compose file if the target is configured with a manual proxy", func(t *testing.T) {
			target := fixture.Target(fixture.WithProviderConfig(docker.Data{}))
			productionConfig := domain.NewEnvironmentConfig(target.ID())
			productionConfig.HasEnvironmentVariables(domain.ServicesEnv{
				"app": domain.EnvVars{
					"DSN": "postgres://prodapp:passprod@db/app?sslmode=disable",
				},
				"db": domain.EnvVars{
					"POSTGRES_USER":     "prodapp",
					"POSTGRES_PASSWORD": "passprod",
				},
			})
			app := fixture.App(
				fixture.WithAppName("my-app"),
				fixture.WithEnvironmentConfig(
					productionConfig,
					domain.NewEnvironmentConfig(target.ID()),
				),
			)
			deployment := fixture.Deployment(
				fixture.FromApp(app),
				fixture.ForEnvironment(domain.Production),
				fixture.WithSourceData(raw.Data(`services:
  sidecar:
    image: traefik/whoami
    profiles:
      - production
  app:
    restart: unless-stopped
    build: .
    environment:
      - DSN=postgres://app:apppa55word@db/app?sslmode=disable
    depends_on:
      - db
    ports:
      - "8080:8080"
      - "8081:8081/udp"
      - "8082:8082"
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
    ports:
      - "5432:5432/tcp"
volumes:
  dbdata:`)),
			)
			appIdLower := strings.ToLower(string(app.ID()))

			// Prepare the build
			opts := config.Default(config.WithTestDefaults())
			artifactManager := artifact.NewLocal(opts, logger)
			deploymentContext, err := artifactManager.PrepareBuild(context.Background(), deployment)
			assert.Nil(t, err)
			assert.Nil(t, raw.New().Fetch(context.Background(), deploymentContext, deployment))
			defer deploymentContext.Logger().Close()
			provider, mock := arrange(opts)

			services, err := provider.Deploy(context.Background(), deploymentContext, deployment, target, nil)

			assert.Nil(t, err)
			assert.HasLength(t, 1, mock.ups)
			assert.HasLength(t, 3, services)

			assert.Equal(t, "app", services[0].Name())
			assert.Equal(t, "db", services[1].Name())
			assert.Equal(t, "sidecar", services[2].Name())

			assert.HasLength(t, 0, services.Entrypoints())

			project := mock.ups[0].project
			expectedProjectName := fmt.Sprintf("%s-%s-%s", deployment.Config().AppName(), deployment.Config().Environment(), appIdLower)
			assert.Equal(t, expectedProjectName, project.Name)
			assert.Equal(t, 3, len(project.Services))

			for _, service := range project.Services {
				switch service.Name {
				case "sidecar":
					assert.Equal(t, "traefik/whoami", service.Image)
					assert.HasLength(t, 0, service.Ports)
					assert.DeepEqual(t, types.MappingWithEquals{}, service.Environment)
					assert.DeepEqual(t, types.Labels{
						docker.AppLabel:         string(deployment.ID().AppID()),
						docker.TargetLabel:      string(target.ID()),
						docker.EnvironmentLabel: string(deployment.Config().Environment()),
					}, service.Labels)
					assert.DeepEqual(t, map[string]*types.ServiceNetworkConfig{
						"default": nil,
					}, service.Networks)
				case "app":
					dsn := deployment.Config().EnvironmentVariablesFor("app").MustGet()["DSN"]

					assert.Equal(t, fmt.Sprintf("%s-%s/app:%s", deployment.Config().AppName(), appIdLower, deployment.Config().Environment()), service.Image)
					assert.Equal(t, types.RestartPolicyUnlessStopped, service.Restart)
					assert.DeepEqual(t, types.Labels{
						docker.AppLabel:         string(deployment.ID().AppID()),
						docker.TargetLabel:      string(target.ID()),
						docker.EnvironmentLabel: string(deployment.Config().Environment()),
					}, service.Labels)

					assert.DeepEqual(t, []types.ServicePortConfig{
						{
							Protocol:  "tcp",
							Mode:      "ingress",
							Target:    8080,
							Published: "8080",
						},
						{
							Protocol:  "udp",
							Mode:      "ingress",
							Target:    8081,
							Published: "8081",
						},
						{
							Protocol:  "tcp",
							Mode:      "ingress",
							Target:    8082,
							Published: "8082",
						},
					}, service.Ports)
					assert.DeepEqual(t, types.MappingWithEquals{
						"DSN": &dsn,
					}, service.Environment)
					assert.DeepEqual(t, map[string]*types.ServiceNetworkConfig{
						"default": nil,
					}, service.Networks)
				case "db":
					postgresUser := deployment.Config().EnvironmentVariablesFor("db").MustGet()["POSTGRES_USER"]
					postgresPassword := deployment.Config().EnvironmentVariablesFor("db").MustGet()["POSTGRES_PASSWORD"]

					assert.Equal(t, "postgres:14-alpine", service.Image)
					assert.Equal(t, types.RestartPolicyUnlessStopped, service.Restart)
					assert.DeepEqual(t, types.Labels{
						docker.AppLabel:         string(deployment.ID().AppID()),
						docker.TargetLabel:      string(target.ID()),
						docker.EnvironmentLabel: string(deployment.Config().Environment()),
					}, service.Labels)
					assert.DeepEqual(t, []types.ServicePortConfig{
						{
							Protocol:  "tcp",
							Mode:      "ingress",
							Target:    5432,
							Published: "5432",
						},
					}, service.Ports)
					assert.DeepEqual(t, types.MappingWithEquals{
						"POSTGRES_USER":     &postgresUser,
						"POSTGRES_PASSWORD": &postgresPassword,
					}, service.Environment)
					assert.DeepEqual(t, map[string]*types.ServiceNetworkConfig{
						"default": nil,
					}, service.Networks)
					assert.DeepEqual(t, []types.ServiceVolumeConfig{
						{
							Type:   types.VolumeTypeVolume,
							Source: "dbdata",
							Target: "/var/lib/postgresql/data",
							Volume: &types.ServiceVolumeVolume{},
						},
					}, service.Volumes)
				default:
					t.Fatalf("unexpected service %s", service.Name)
				}
			}

			assert.DeepEqual(t, types.Networks{
				"default": {
					Name: expectedProjectName + "_default",
					Labels: types.Labels{
						docker.TargetLabel:      string(target.ID()),
						docker.AppLabel:         string(deployment.Config().AppID()),
						docker.EnvironmentLabel: string(deployment.Config().Environment()),
					},
				},
			}, project.Networks)
			assert.DeepEqual(t, types.Volumes{
				"dbdata": {
					Name: expectedProjectName + "_dbdata",
					Labels: types.Labels{
						docker.TargetLabel:      string(target.ID()),
						docker.AppLabel:         string(deployment.Config().AppID()),
						docker.EnvironmentLabel: string(deployment.Config().Environment()),
					},
				},
			}, project.Volumes)

			assert.DeepEqual(t, filters.NewArgs(
				filters.Arg("dangling", "true"),
				filters.Arg("label", fmt.Sprintf("%s=%s", docker.AppLabel, deployment.ID().AppID())),
				filters.Arg("label", fmt.Sprintf("%s=%s", docker.TargetLabel, target.ID())),
				filters.Arg("label", fmt.Sprintf("%s=%s", docker.EnvironmentLabel, deployment.Config().Environment())),
			), mock.pruneFilters)
		})
	})

}

func sortedPorts(ports []types.ServicePortConfig) []types.ServicePortConfig {
	slices.SortFunc(ports, docker.ServicePortSortFunc)
	return ports
}

type (
	dockerMockService struct {
		api.Service
		command.Cli
		containers   map[string]types.ServiceConfig
		ups          []up
		downs        []down
		pruneFilters filters.Args
	}

	dockerMockCli struct {
		client.APIClient
		parent *dockerMockService
	}

	up struct {
		project *types.Project
		options api.UpOptions
	}

	down struct {
		projectName string
		options     api.DownOptions
	}
)

func newMockService() *dockerMockService {
	return &dockerMockService{
		containers: make(map[string]types.ServiceConfig),
	}
}

func (c *dockerMockService) Up(ctx context.Context, project *types.Project, options api.UpOptions) error {
	for _, service := range project.Services {
		if service.ContainerName != "" {
			c.containers[service.ContainerName] = service
		}
	}

	c.ups = append(c.ups, up{
		project: project,
		options: options,
	})
	return nil
}

func (c *dockerMockService) Down(ctx context.Context, projectName string, options api.DownOptions) error {
	c.downs = append(c.downs, down{
		projectName: projectName,
		options:     options,
	})
	return nil
}

func (d *dockerMockService) Client() client.APIClient {
	return &dockerMockCli{parent: d}
}

func (d *dockerMockService) Apply(ops ...command.CLIOption) error {
	return nil
}

func (d *dockerMockCli) Close() error { return nil }

func (d *dockerMockCli) ContainerInspect(_ context.Context, containerName string) (dockertypes.ContainerJSON, error) {
	container, found := d.parent.containers[containerName]

	if !found {
		return dockertypes.ContainerJSON{}, errors.New("not found")
	}

	result := dockertypes.ContainerJSON{
		NetworkSettings: &dockertypes.NetworkSettings{
			NetworkSettingsBase: dockertypes.NetworkSettingsBase{
				Ports: nat.PortMap{},
			},
		},
	}

	// For this mock, only assign the host port to the target one
	for _, port := range container.Ports {
		result.NetworkSettings.Ports[nat.Port(fmt.Sprintf("%d/%s", port.Target, port.Protocol))] = []nat.PortBinding{
			{HostPort: strconv.FormatUint(uint64(port.Target), 10)},
		}
	}

	return result, nil
}

func (d *dockerMockCli) ImagesPrune(_ context.Context, criteria filters.Args) (image.PruneReport, error) {
	d.parent.pruneFilters = criteria
	return image.PruneReport{}, nil
}
