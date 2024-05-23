package docker_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"

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
	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

type options interface {
	artifact.LocalOptions
}

func Test_Provider(t *testing.T) {
	logger := must.Panic(log.NewLogger())

	sut := func(opts options) (docker.Docker, *dockerMockService) {
		mock := newMockService()

		t.Cleanup(func() {
			os.RemoveAll(opts.DataDir())
		})

		return docker.New(logger, docker.WithDockerAndCompose(mock, mock)), mock
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

		provider, _ := sut(config.Default(config.WithTestDefaults()))

		for _, tt := range tests {
			t.Run(fmt.Sprintf("%v", tt.payload), func(t *testing.T) {
				data, err := provider.Prepare(context.Background(), tt.payload, tt.existing...)

				testutil.IsNil(t, err)
				testutil.IsTrue(t, data.Equals(tt.expected))
			})
		}
	})

	t.Run("should setup a new non-ssl target without custom entrypoints", func(t *testing.T) {
		target := createTarget("http://docker.localhost")
		targetIdLower := strings.ToLower(string(target.ID()))

		provider, mock := sut(config.Default(config.WithTestDefaults()))

		assigned, err := provider.Setup(context.Background(), target)

		testutil.IsNil(t, err)
		testutil.DeepEquals(t, domain.TargetEntrypointsAssigned{}, assigned)
		testutil.HasLength(t, mock.ups, 1)
		testutil.DeepEquals(t, &types.Project{
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

	t.Run("should setup a new ssl target without custom entrypoints", func(t *testing.T) {
		target := createTarget("https://docker.localhost")
		targetIdLower := strings.ToLower(string(target.ID()))

		provider, mock := sut(config.Default(config.WithTestDefaults()))

		assigned, err := provider.Setup(context.Background(), target)

		testutil.IsNil(t, err)
		testutil.DeepEquals(t, domain.TargetEntrypointsAssigned{}, assigned)
		testutil.HasLength(t, mock.ups, 1)
		testutil.DeepEquals(t, &types.Project{
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

	t.Run("should setup a target with custom entrypoints by finding available ports", func(t *testing.T) {
		target := createTarget("http://docker.localhost")
		targetIdLower := strings.ToLower(string(target.ID()))
		depl := createDeployment(target.ID(), "")

		service := depl.Config().NewService("app", "")
		tcp := service.AddTCPEntrypoint(5432)
		udp := service.AddUDPEntrypoint(5433)

		target.ExposeEntrypoints(depl.ID().AppID(), depl.Config().Environment(), domain.Services{service})

		provider, mock := sut(config.Default(config.WithTestDefaults()))

		assigned, err := provider.Setup(context.Background(), target)

		testutil.IsNil(t, err)
		testutil.HasLength(t, mock.ups, 2)
		testutil.HasLength(t, mock.downs, 1)

		tcpPort := assigned[depl.ID().AppID()][depl.Config().Environment()][tcp.Name()]
		udpPort := assigned[depl.ID().AppID()][depl.Config().Environment()][udp.Name()]

		testutil.NotEquals(t, 0, tcpPort)
		testutil.NotEquals(t, 0, udpPort)

		testutil.DeepEquals(t, &types.Project{
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

	t.Run("should setup a target with custom entrypoints by using provided ports if any", func(t *testing.T) {
		target := createTarget("http://docker.localhost")
		targetIdLower := strings.ToLower(string(target.ID()))
		depl := createDeployment(target.ID(), "")

		service := depl.Config().NewService("app", "")
		tcp := service.AddTCPEntrypoint(5432)
		udp := service.AddUDPEntrypoint(5433)

		target.ExposeEntrypoints(depl.ID().AppID(), depl.Config().Environment(), domain.Services{service})
		target.Configured(target.CurrentVersion(), domain.TargetEntrypointsAssigned{
			depl.ID().AppID(): {
				depl.Config().Environment(): {
					tcp.Name(): 5432,
					udp.Name(): 5433,
				},
			},
		}, nil)

		newTcp := service.AddTCPEntrypoint(5434)
		newUdp := service.AddUDPEntrypoint(5435)
		target.ExposeEntrypoints(depl.ID().AppID(), depl.Config().Environment(), domain.Services{service})

		provider, mock := sut(config.Default(config.WithTestDefaults()))

		assigned, err := provider.Setup(context.Background(), target)

		testutil.IsNil(t, err)
		testutil.HasLength(t, mock.ups, 2)
		testutil.HasLength(t, mock.downs, 1)
		testutil.Equals(t, 2, len(assigned[depl.ID().AppID()][depl.Config().Environment()]))

		tcpPort := assigned[depl.ID().AppID()][depl.Config().Environment()][newTcp.Name()]
		udpPort := assigned[depl.ID().AppID()][depl.Config().Environment()][newUdp.Name()]

		testutil.NotEquals(t, 0, tcpPort)
		testutil.NotEquals(t, 0, udpPort)

		testutil.DeepEquals(t, &types.Project{
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

	t.Run("should expose services from a compose file", func(t *testing.T) {
		target := createTarget("http://docker.localhost")
		depl := createDeployment(target.ID(), `services:
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
  dbdata:`)
		appIdLower := strings.ToLower(string(depl.ID().AppID()))

		// Prepare the build
		opts := config.Default(config.WithTestDefaults())
		artifactManager := artifact.NewLocal(opts, logger)
		ctx, err := artifactManager.PrepareBuild(context.Background(), depl)
		testutil.IsNil(t, err)
		testutil.IsNil(t, raw.New().Fetch(context.Background(), ctx, depl))

		provider, mock := sut(opts)

		services, err := provider.Deploy(context.Background(), ctx, depl, target, nil)

		testutil.IsNil(t, err)
		testutil.HasLength(t, mock.ups, 1)
		testutil.HasLength(t, services, 3)

		testutil.Equals(t, "app", services[0].Name())
		testutil.Equals(t, "db", services[1].Name())
		testutil.Equals(t, "sidecar", services[2].Name())

		entrypoints := services.Entrypoints()
		testutil.HasLength(t, entrypoints, 4)
		testutil.Equals(t, 8080, entrypoints[0].Port())
		testutil.Equals(t, "http", entrypoints[0].Router())
		testutil.Equals(t, string(depl.Config().AppName()), entrypoints[0].Subdomain().Get(""))
		testutil.Equals(t, 8081, entrypoints[1].Port())
		testutil.Equals(t, "udp", entrypoints[1].Router())
		testutil.Equals(t, 8082, entrypoints[2].Port())
		testutil.Equals(t, "http", entrypoints[2].Router())
		testutil.Equals(t, string(depl.Config().AppName()), entrypoints[2].Subdomain().Get(""))
		testutil.Equals(t, 5432, entrypoints[3].Port())
		testutil.Equals(t, "tcp", entrypoints[3].Router())

		project := mock.ups[0].project
		expectedProjectName := fmt.Sprintf("%s-%s-%s", depl.Config().AppName(), depl.Config().Environment(), appIdLower)
		expectedGatewayNetworkName := "seelf-gateway-" + strings.ToLower(string(target.ID()))
		testutil.Equals(t, expectedProjectName, project.Name)
		testutil.Equals(t, 3, len(project.Services))

		for _, service := range project.Services {
			switch service.Name {
			case "sidecar":
				testutil.Equals(t, "traefik/whoami", service.Image)
				testutil.HasLength(t, service.Ports, 0)
				testutil.DeepEquals(t, types.MappingWithEquals{}, service.Environment)
				testutil.DeepEquals(t, types.Labels{
					docker.AppLabel:         string(depl.ID().AppID()),
					docker.TargetLabel:      string(target.ID()),
					docker.EnvironmentLabel: string(depl.Config().Environment()),
				}, service.Labels)
				testutil.DeepEquals(t, map[string]*types.ServiceNetworkConfig{
					"default": nil,
				}, service.Networks)
			case "app":
				httpEntrypointName := string(entrypoints[0].Name())
				udpEntrypointName := string(entrypoints[1].Name())
				customHttpEntrypointName := string(entrypoints[2].Name())
				dsn := depl.Config().EnvironmentVariablesFor("app").MustGet()["DSN"]

				testutil.Equals(t, fmt.Sprintf("%s-%s/app:%s", depl.Config().AppName(), appIdLower, depl.Config().Environment()), service.Image)
				testutil.Equals(t, types.RestartPolicyUnlessStopped, service.Restart)
				testutil.DeepEquals(t, types.Labels{
					docker.AppLabel:         string(depl.ID().AppID()),
					docker.TargetLabel:      string(target.ID()),
					docker.EnvironmentLabel: string(depl.Config().Environment()),
					docker.SubdomainLabel:   string(depl.Config().AppName()),
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

				testutil.HasLength(t, service.Ports, 0)
				testutil.DeepEquals(t, types.MappingWithEquals{
					"DSN": &dsn,
				}, service.Environment)
				testutil.DeepEquals(t, map[string]*types.ServiceNetworkConfig{
					"default":                  nil,
					expectedGatewayNetworkName: nil,
				}, service.Networks)
			case "db":
				entrypointName := string(entrypoints[3].Name())
				postgresUser := depl.Config().EnvironmentVariablesFor("db").MustGet()["POSTGRES_USER"]
				postgresPassword := depl.Config().EnvironmentVariablesFor("db").MustGet()["POSTGRES_PASSWORD"]

				testutil.Equals(t, "postgres:14-alpine", service.Image)
				testutil.Equals(t, types.RestartPolicyUnlessStopped, service.Restart)
				testutil.DeepEquals(t, types.Labels{
					docker.AppLabel:         string(depl.ID().AppID()),
					docker.TargetLabel:      string(target.ID()),
					docker.EnvironmentLabel: string(depl.Config().Environment()),
					fmt.Sprintf("traefik.tcp.routers.%s.rule", entrypointName):                      "HostSNI(`*`)",
					docker.CustomEntrypointsLabel:                                                   "true",
					fmt.Sprintf("traefik.tcp.routers.%s.entrypoints", entrypointName):               entrypointName,
					fmt.Sprintf("traefik.tcp.routers.%s.service", entrypointName):                   entrypointName,
					fmt.Sprintf("traefik.tcp.services.%s.loadbalancer.server.port", entrypointName): "5432",
				}, service.Labels)
				testutil.HasLength(t, service.Ports, 0)
				testutil.DeepEquals(t, types.MappingWithEquals{
					"POSTGRES_USER":     &postgresUser,
					"POSTGRES_PASSWORD": &postgresPassword,
				}, service.Environment)
				testutil.DeepEquals(t, map[string]*types.ServiceNetworkConfig{
					"default":                  nil,
					expectedGatewayNetworkName: nil,
				}, service.Networks)
				testutil.DeepEquals(t, []types.ServiceVolumeConfig{
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

		testutil.DeepEquals(t, types.Networks{
			"default": {
				Name: expectedProjectName + "_default",
				Labels: types.Labels{
					docker.TargetLabel:      string(target.ID()),
					docker.AppLabel:         string(depl.Config().AppID()),
					docker.EnvironmentLabel: string(depl.Config().Environment()),
				},
			},
			expectedGatewayNetworkName: {
				Name:     expectedGatewayNetworkName,
				External: true,
			},
		}, project.Networks)
		testutil.DeepEquals(t, types.Volumes{
			"dbdata": {
				Name: expectedProjectName + "_dbdata",
				Labels: types.Labels{
					docker.TargetLabel:      string(target.ID()),
					docker.AppLabel:         string(depl.Config().AppID()),
					docker.EnvironmentLabel: string(depl.Config().Environment()),
				},
			},
		}, project.Volumes)

		testutil.DeepEquals(t, filters.NewArgs(
			filters.Arg("dangling", "true"),
			filters.Arg("label", fmt.Sprintf("%s=%s", docker.AppLabel, depl.ID().AppID())),
			filters.Arg("label", fmt.Sprintf("%s=%s", docker.TargetLabel, target.ID())),
			filters.Arg("label", fmt.Sprintf("%s=%s", docker.EnvironmentLabel, depl.Config().Environment())),
		), mock.pruneFilters)
	})
}

func createTarget(url string) domain.Target {
	return must.Panic(domain.NewTarget(
		"a target",
		domain.NewTargetUrlRequirement(must.Panic(domain.UrlFrom(url)), true),
		domain.NewProviderConfigRequirement(docker.Data{}, true),
		"uid",
	))
}

func createDeployment(target domain.TargetID, data string) domain.Deployment {
	productionConfig := domain.NewEnvironmentConfig(target)
	productionConfig.HasEnvironmentVariables(domain.ServicesEnv{
		"app": domain.EnvVars{
			"DSN": "postgres://prodapp:passprod@db/app?sslmode=disable",
		},
		"db": domain.EnvVars{
			"POSTGRES_USER":     "prodapp",
			"POSTGRES_PASSWORD": "passprod",
		},
	})

	app := must.Panic(domain.NewApp(
		"my-app",
		domain.NewEnvironmentConfigRequirement(productionConfig, true, true),
		domain.NewEnvironmentConfigRequirement(domain.NewEnvironmentConfig(target), true, true),
		"uid",
	))

	return must.Panic(app.NewDeployment(1, raw.Data(data), domain.Production, "uid"))
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

func (d *dockerMockCli) ImagesPrune(_ context.Context, criteria filters.Args) (dockertypes.ImagesPruneReport, error) {
	d.parent.pruneFilters = criteria
	return dockertypes.ImagesPruneReport{}, nil
}

// func (d *dockerMockService) ContainerList(context.Context, container.ListOptions) ([]dockertypes.Container, error) {
// 	return nil, nil
// }

// func (d *dockerMockService) VolumeList(context.Context, volume.ListOptions) (volume.ListResponse, error) {
// 	return volume.ListResponse{}, nil
// }

// func (d *dockerMockService) NetworkList(context.Context, dockertypes.NetworkListOptions) ([]dockertypes.NetworkResource, error) {
// 	return nil, nil
// }

// func (d *dockerMockService) ImageList(context.Context, image.ListOptions) ([]image.Summary, error) {
// 	return nil, nil
// }
