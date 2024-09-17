package docker

import (
	"context"
	"slices"
	"strings"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/compose-spec/compose-go/v2/types"
	"github.com/docker/compose/v2/pkg/api"
	"github.com/docker/go-connections/nat"
)

const (
	httpMainEntryPoint      = "http"
	portsFinderStartingPort = 8080
)

type (
	// Builder used to create a compose project with everything needed to deploy
	// the proxy used to expose application entrypoints.
	// It will handle the assignment of new entrypoints ports if needed.
	proxyProjectBuilder struct {
		client           *client
		target           string
		host             string
		projectName      string
		networkName      string
		certResolverName string
		entrypoints      domain.TargetEntrypoints
		assigned         domain.TargetEntrypointsAssigned
		newEntrypoints   []entrypointDefinition
		newPorts         []types.ServicePortConfig
		labels           types.Labels
		project          *types.Project
		proxy            types.ServiceConfig
	}

	entrypointDefinition struct {
		app  domain.AppID
		env  domain.Environment
		name domain.EntrypointName
		key  nat.Port
	}
)

func newProxyProjectBuilder(client *client, target domain.Target) *proxyProjectBuilder {
	id := target.ID()
	idLower := strings.ToLower(string(id))
	url := target.Url().MustGet()

	b := &proxyProjectBuilder{
		client:      client,
		target:      string(id),
		host:        url.Host(),
		entrypoints: target.CustomEntrypoints(),
		assigned:    make(domain.TargetEntrypointsAssigned),
		networkName: targetPublicNetworkName(id),
		projectName: targetProjectName(id),
		labels:      types.Labels{TargetLabel: string(id)},
	}

	if url.UseSSL() {
		b.certResolverName = "seelf-resolver-" + idLower
	}

	return b
}

func (b *proxyProjectBuilder) Build(ctx context.Context) (*types.Project, domain.TargetEntrypointsAssigned, error) {
	b.createComposeProject()

	if err := b.prepareCustomEntrypoints(); err != nil {
		return nil, nil, err
	}

	if err := b.resolveNewEntrypoints(ctx); err != nil {
		return nil, nil, err
	}

	// Append the now fully loaded proxy service to the project
	b.project.Services = types.Services{
		b.proxy.Name: b.proxy,
	}

	b.normalize()

	return b.project, b.assigned, nil
}

func (b *proxyProjectBuilder) createComposeProject() {
	b.project = &types.Project{
		Name: b.projectName,
		Networks: types.Networks{
			"default": types.NetworkConfig{
				Name:   b.networkName,
				Labels: b.labels,
			},
		},
	}

	b.proxy = types.ServiceConfig{
		Name:    "proxy",
		Labels:  b.labels,
		Image:   "traefik:v2.11",
		Restart: types.RestartPolicyUnlessStopped,
		Command: types.ShellCommand{
			// "--api.insecure=true",
			"--providers.docker",
			"--providers.docker.network=" + b.networkName,
			"--providers.docker.constraints=(Label(`" + TargetLabel + "`, `" + b.target + "`) && (Label(`" + CustomEntrypointsLabel + "`, `true`) || LabelRegex(`" + SubdomainLabel + "`, `.+`))) || Label(`" + ExposedLabel + "`, `true`)",
			"--providers.docker.defaultrule=Host(`{{ index .Labels " + `"` + SubdomainLabel + `"` + "}}." + b.host + "`)",
			"--entrypoints." + httpMainEntryPoint + ".address=:80",
		},
		Ports: []types.ServicePortConfig{
			{Target: 80, Published: "80"},
			// {Target: 8080, Published: "8081"},
		},
		Volumes: []types.ServiceVolumeConfig{
			{Type: types.VolumeTypeBind, Source: "/var/run/docker.sock", Target: "/var/run/docker.sock"},
		},
		CustomLabels: getProjectCustomLabels(b.projectName, "proxy", ""),
	}

	if b.certResolverName != "" {
		b.proxy.Command = append(b.proxy.Command[:len(b.proxy.Command)-1],
			"--entrypoints.insecure.address=:80",
			"--entrypoints.insecure.http.redirections.entryPoint.to="+httpMainEntryPoint,
			"--entrypoints.insecure.http.redirections.entryPoint.scheme=https",
			"--entrypoints."+httpMainEntryPoint+".address=:443",
			"--certificatesresolvers."+b.certResolverName+".acme.tlschallenge=true",
			"--certificatesresolvers."+b.certResolverName+".acme.storage=/letsencrypt/acme.json",
			"--entrypoints."+httpMainEntryPoint+".http.tls.certresolver="+b.certResolverName,
		)

		b.proxy.Ports = append(b.proxy.Ports, types.ServicePortConfig{
			Target: 443, Published: "443",
		})

		b.proxy.Volumes = append(b.proxy.Volumes, types.ServiceVolumeConfig{
			Type:   types.VolumeTypeVolume,
			Source: "letsencrypt",
			Target: "/letsencrypt",
		})

		b.project.Volumes = types.Volumes{
			"letsencrypt": types.VolumeConfig{
				Name:   b.projectName + "_letsencrypt",
				Labels: b.labels,
			},
		}
	}
}

func (b *proxyProjectBuilder) prepareCustomEntrypoints() error {
	var err error

	// Apply existing entrypoints and keep track of new ones
	for appid, envs := range b.entrypoints {
		for env, entrypoints := range envs {
			for name, entry := range entrypoints {
				// Port already known
				if port, hasValue := entry.TryGet(); hasValue {
					b.addEntrypoint(name, port)
					continue
				}

				newEntrypoint := entrypointDefinition{
					app:  appid,
					env:  env,
					name: name,
				}

				proto := name.Protocol()
				port := domain.Port(portsFinderStartingPort + len(b.newEntrypoints))

				b.newPorts = append(b.newPorts, types.ServicePortConfig{
					Target:   port.Uint32(),
					Protocol: proto,
				})

				newEntrypoint.key, err = nat.NewPort(proto, port.String())

				if err != nil {
					return err
				}

				b.newEntrypoints = append(b.newEntrypoints, newEntrypoint)
			}
		}
	}

	return nil
}

func (b *proxyProjectBuilder) resolveNewEntrypoints(ctx context.Context) error {
	if len(b.newEntrypoints) == 0 {
		return nil
	}

	var (
		err           error
		projectName   = b.projectName + "-ports-finder"
		containerName = projectName + "_app"
	)

	finderProject := &types.Project{
		Name: projectName,
		Services: types.Services{
			"app": types.ServiceConfig{
				ContainerName: containerName,
				Image:         "traefik/whoami",
				Labels:        b.labels,
				Name:          "app",
				Ports:         b.newPorts,
				CustomLabels:  getProjectCustomLabels(projectName, "app", ""),
				Networks: map[string]*types.ServiceNetworkConfig{
					"default": nil,
				},
			},
		},
		Networks: types.Networks{
			"default": types.NetworkConfig{
				Name:   projectName + "_network",
				Labels: b.labels,
			},
		},
	}

	if err := b.client.compose.Up(ctx, finderProject, api.UpOptions{
		Create: api.CreateOptions{
			RemoveOrphans: true,
			Recreate:      api.RecreateForce,
		},
		Start: api.StartOptions{
			Wait: true,
		},
	}); err != nil {
		return err
	}

	info, err := b.client.api.ContainerInspect(ctx, containerName)

	if err != nil {
		return err
	}

	for _, newPort := range b.newEntrypoints {
		port := info.NetworkSettings.Ports[newPort.key]
		parsedPort, err := domain.ParsePort(port[0].HostPort)

		if err != nil {
			return err
		}

		b.addEntrypoint(newPort.name, parsedPort)
		b.assigned.Set(newPort.app, newPort.env, newPort.name, parsedPort)
	}

	return b.client.compose.Down(ctx, projectName, api.DownOptions{})
}

func (b *proxyProjectBuilder) addEntrypoint(name domain.EntrypointName, port domain.Port) {
	var (
		published = port.String()
		proto     = name.Protocol()
	)

	b.proxy.Command = append(b.proxy.Command,
		"--entrypoints."+string(name)+".address=:"+published+"/"+proto)
	b.proxy.Ports = append(b.proxy.Ports, types.ServicePortConfig{
		Target:    uint32(port),
		Published: published,
		Protocol:  proto,
	})
}

// Normalize command and ports to make sure the service hash will not changed.
func (b *proxyProjectBuilder) normalize() {
	for name := range b.project.Services {
		slices.Sort(b.project.Services[name].Command)
		slices.SortFunc(b.project.Services[name].Ports, ServicePortSortFunc)
	}
}

// Sort function for service ports
func ServicePortSortFunc(a, b types.ServicePortConfig) int {
	return int(a.Target) - int(b.Target)
}

// Retrieve the network name of a specific target
func targetPublicNetworkName(id domain.TargetID) string {
	return "seelf-gateway-" + strings.ToLower(string(id))
}

// Retrieve the project name of a specific target
func targetProjectName(id domain.TargetID) string {
	return "seelf-internal-" + strings.ToLower(string(id))
}
