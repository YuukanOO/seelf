package docker

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/compose-spec/compose-go/v2/cli"
	"github.com/compose-spec/compose-go/v2/interpolation"
	"github.com/compose-spec/compose-go/v2/loader"
	"github.com/compose-spec/compose-go/v2/tree"
	"github.com/compose-spec/compose-go/v2/types"
	"github.com/docker/go-connections/nat"
	"golang.org/x/exp/maps"
)

type deploymentProjectBuilder struct {
	exposedManually             bool
	sourceDir                   string
	composePath                 string
	networkName                 string
	services                    domain.Services
	project                     *types.Project
	config                      domain.DeploymentConfig
	logger                      domain.DeploymentLogger
	labels                      types.Labels
	isDefaultSubdomainAvailable bool
	routersByPort               map[string]domain.Router
}

func newDeploymentProjectBuilder(
	ctx domain.DeploymentContext,
	deployment domain.Deployment,
	exposedManually bool,
) *deploymentProjectBuilder {
	config := deployment.Config()

	return &deploymentProjectBuilder{
		exposedManually:             exposedManually,
		isDefaultSubdomainAvailable: true,
		sourceDir:                   ctx.BuildDirectory(),
		config:                      config,
		networkName:                 targetPublicNetworkName(config.Target()),
		logger:                      ctx.Logger(),
		routersByPort:               make(map[string]domain.Router),
		labels: types.Labels{
			AppLabel:         string(deployment.ID().AppID()),
			TargetLabel:      string(config.Target()),
			EnvironmentLabel: string(config.Environment()),
		},
	}
}

func (b *deploymentProjectBuilder) Build(ctx context.Context) (*types.Project, domain.Services, error) {
	if err := b.findComposeFile(); err != nil {
		return nil, nil, err
	}

	if err := b.loadProject(ctx); err != nil {
		return nil, nil, err
	}

	b.transform()

	return b.project, b.services, nil
}

func (b *deploymentProjectBuilder) findComposeFile() error {
	var (
		envStr               = string(b.config.Environment())
		serviceFilesAffinity = []string{
			"compose.seelf." + envStr + ".yml",
			"compose.seelf." + envStr + ".yaml",
			"docker-compose.seelf." + envStr + ".yml",
			"docker-compose.seelf." + envStr + ".yaml",
			"compose." + envStr + ".yml",
			"compose." + envStr + ".yaml",
			"docker-compose." + envStr + ".yml",
			"docker-compose." + envStr + ".yaml",
			"compose.seelf.yml",
			"compose.seelf.yaml",
			"docker-compose.seelf.yml",
			"docker-compose.seelf.yaml",
			"compose.yml",
			"compose.yaml",
			"docker-compose.yml",
			"docker-compose.yaml",
		}
	)

	for _, file := range serviceFilesAffinity {
		servicePath := filepath.Join(b.sourceDir, file)
		_, err := os.Stat(servicePath)

		if os.IsNotExist(err) {
			continue
		}

		if err == nil {
			b.composePath = servicePath
			return nil
		}

		b.logger.Error(err)
		return ErrOpenComposeFileFailed
	}

	b.logger.Error(fmt.Errorf("could not find a valid compose file, tried in the following order:\n\t%s", strings.Join(serviceFilesAffinity, "\n\t")))

	return ErrOpenComposeFileFailed
}

func (b *deploymentProjectBuilder) loadProject(ctx context.Context) error {
	b.logger.Stepf("reading project from %s", b.composePath)

	loaders := []cli.ProjectOptionsFn{
		cli.WithName(b.config.ProjectName()),
		cli.WithNormalization(true),
		cli.WithProfiles([]string{string(b.config.Environment())}),
	}

	if !b.exposedManually {
		loaders = append(loaders, cli.WithLoadOptions(func(o *loader.Options) {
			o.Interpolate = &interpolation.Options{
				TypeCastMapping: map[tree.Path]interpolation.Cast{
					"services.*.ports.[]": func(value string) (any, error) {
						return value, b.parsePortDefinition(value)
					},
				},
			}
		}))
	}

	opts, err := cli.NewProjectOptions([]string{b.composePath}, loaders...)

	if err != nil {
		b.logger.Error(err)
		return ErrLoadProjectFailed
	}

	b.project, err = cli.ProjectFromOptions(ctx, opts)

	if err != nil {
		b.logger.Error(err)
		return ErrLoadProjectFailed
	}

	for i, s := range b.project.Services {
		s.CustomLabels = getProjectCustomLabels(b.project.Name, s.Name, b.project.WorkingDir, b.project.ComposeFiles...)
		b.project.Services[i] = s
	}

	return nil
}

func (b *deploymentProjectBuilder) transform() {
	b.logger.Stepf("configuring seelf docker project for environment: %s", b.config.Environment())

	if len(b.project.DisabledServices) > 0 {
		b.logger.Infof("some services have been disabled by the %s profile: %s", b.config.Environment(), strings.Join(maps.Keys(b.project.DisabledServices), ", "))
		b.project.DisabledServices = nil // Reset the list of disabled services or orphans created for an old profile will not be deleted
	}

	// Let's transform the project to expose needed services
	// Here ServiceNames sort the services by alphabetical order so we don't have to
	for _, name := range b.project.ServiceNames() {
		serviceDefinition := b.project.Services[name]
		service := b.config.NewService(serviceDefinition.Name, serviceDefinition.Image)
		serviceName := service.Name()

		if serviceDefinition.Restart == "" {
			b.logger.Warnf("no restart policy sets for service %s, the service will not be restarted automatically", serviceName)
		}

		// If there's an image to build, force it (same as --build in the docker compose cli)
		if serviceDefinition.Build != nil {
			serviceDefinition.Image = service.Image() // Since the image name may have been generated, override it
			serviceDefinition.PullPolicy = types.PullPolicyBuild
			serviceDefinition.Build.Labels = appendLabels(serviceDefinition.Build.Labels, b.labels)
		}

		// Attach environment variables if any
		servicesEnv := b.config.EnvironmentVariablesFor(serviceName)

		if vars, hasVars := servicesEnv.TryGet(); hasVars {
			envNames := make([]string, 0, len(vars))

			for name, value := range vars {
				localValue := value // Copy the value to avoid the loop to use the same pointer
				serviceDefinition.Environment[name] = &localValue
				envNames = append(envNames, name)
			}

			b.logger.Infof("using %s environment variable(s) for service %s", strings.Join(envNames, ", "), serviceName)
		}

		serviceDefinition.Labels = appendLabels(serviceDefinition.Labels, b.labels)

		for _, volume := range serviceDefinition.Volumes {
			if volume.Type == types.VolumeTypeBind {
				b.logger.Warnf("bind mount detected for service %s, this is not supported and your data are not guaranteed to be preserved, use docker volumes instead", serviceName)
			}
		}

		// No ports mapped or manual target, nothing to do
		if b.exposedManually || len(serviceDefinition.Ports) == 0 {
			b.project.Services[serviceName] = serviceDefinition
			b.services = append(b.services, service)
			continue
		}

		var (
			httpMainEntryPointAvailable = true
			entrypoint                  domain.Entrypoint
		)

		for _, portConfig := range serviceDefinition.Ports {
			router, err := b.routerFor(portConfig)

			if err != nil {
				b.logger.Warnf("skipping port exposure: %s", err.Error())
				continue
			}

			switch router {
			case domain.RouterHttp:
				entrypoint = service.AddHttpEntrypoint(b.config, domain.Port(portConfig.Target), domain.HttpEntrypointOptions{
					Managed:             httpMainEntryPointAvailable,
					UseDefaultSubdomain: b.isDefaultSubdomainAvailable,
				})
				httpMainEntryPointAvailable = false
				serviceDefinition.Labels[SubdomainLabel] = entrypoint.Subdomain().MustGet()
			case domain.RouterTcp:
				entrypoint = service.AddTCPEntrypoint(domain.Port(portConfig.Target))
				serviceDefinition.Labels["traefik.tcp.routers."+string(entrypoint.Name())+".rule"] = "HostSNI(`*`)"
			case domain.RouterUdp:
				entrypoint = service.AddUDPEntrypoint(domain.Port(portConfig.Target))
			default:
				b.logger.Warnf("unsupported router type for service %s, the service will not be exposed", serviceName)
				continue
			}

			var (
				entrypointName = string(entrypoint.Name())
				routerName     = string(router)
			)

			if !entrypoint.IsCustom() {
				serviceDefinition.Labels["traefik."+routerName+".routers."+entrypointName+".entrypoints"] = httpMainEntryPoint
				b.isDefaultSubdomainAvailable = false
			} else {
				serviceDefinition.Labels[CustomEntrypointsLabel] = "true"
				serviceDefinition.Labels["traefik."+routerName+".routers."+entrypointName+".entrypoints"] = entrypointName
				b.logger.Infof("using custom entrypoint for service %s (%d/%s)", serviceName, portConfig.Target, portConfig.Protocol)
			}

			serviceDefinition.Labels["traefik."+routerName+".routers."+entrypointName+".service"] = entrypointName
			serviceDefinition.Labels["traefik."+routerName+".services."+entrypointName+".loadbalancer.server.port"] = entrypoint.Port().String()
		}

		serviceDefinition.Ports = []types.ServicePortConfig{} // Remove them since traefik will expose this service

		if serviceDefinition.Networks == nil {
			serviceDefinition.Networks = map[string]*types.ServiceNetworkConfig{}
		}

		serviceDefinition.Networks[b.networkName] = nil // nil here because there's no additional options to give

		// Update the project definition and state
		b.project.Services[serviceName] = serviceDefinition
		b.services = append(b.services, service)
	}

	// Add labels to network and volumes to make it easy to find them
	for name, network := range b.project.Networks {
		network.Labels = appendLabels(network.Labels, b.labels)
		b.project.Networks[name] = network
	}

	for name, volume := range b.project.Volumes {
		volume.Labels = appendLabels(volume.Labels, b.labels)
		b.project.Volumes[name] = volume
	}

	// Append the public seelf network to the project
	if b.exposedManually {
		return
	}

	if b.project.Networks == nil {
		b.project.Networks = types.Networks{}
	}

	b.project.Networks[b.networkName] = types.NetworkConfig{
		Name:     b.networkName,
		External: true,
	}
}

func (b *deploymentProjectBuilder) parsePortDefinition(rawValue string) error {
	explicit := strings.Contains(rawValue, "/")
	ports, _ := nat.ParsePortSpec(rawValue)

	for _, port := range ports {
		// Since we do not know which service this port definition is tied to,
		// (due to how the compose parsing works), we should make sure the port is unique.
		// Host port SHOULD be unique in a compose file or the binding will fail, this is
		// why we dismiss ports without explicit host mapping.
		if port.Binding.HostPort == "" {
			b.logger.Warnf("port %s is missing host port, this is mandatory for seelf, this port will be ignored", port.Port.Port())
			continue
		}

		proto := domain.Router(port.Port.Proto())

		if !explicit {
			proto = domain.RouterHttp
		}

		b.routersByPort[port.Binding.HostPort+":"+port.Port.Port()+"/"+port.Port.Proto()] = proto
	}

	return nil
}

func (b *deploymentProjectBuilder) routerFor(config types.ServicePortConfig) (domain.Router, error) {
	r, found := b.routersByPort[config.Published+":"+strconv.FormatUint(uint64(config.Target), 10)+"/"+config.Protocol]

	if !found {
		return "", fmt.Errorf("could not determine the relevant router type for port %d", config.Target)
	}

	return r, nil
}
