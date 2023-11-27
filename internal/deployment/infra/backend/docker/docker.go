package docker

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/pkg/log"
	"github.com/compose-spec/compose-go/cli"
	"github.com/compose-spec/compose-go/loader"
	"github.com/compose-spec/compose-go/types"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/flags"
	"github.com/docker/compose/v2/pkg/api"
	"github.com/docker/compose/v2/pkg/compose"
	dockertypes "github.com/docker/docker/api/types"
	dockercontainer "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/volume"
)

var (
	ErrLoadProjectFailed     = errors.New("compose_file_malformed")
	ErrOpenComposeFileFailed = errors.New("compose_file_open_failed")
	ErrComposeFailed         = errors.New("compose_failed")
)

const (
	balancerProjectName = "seelf-internal"
	balancerServiceName = "balancer"
	certResolverName    = "seelfresolver"
	publicNetworkName   = "seelf-public"

	AppLabel         = "app.seelf.application"
	EnvironmentLabel = "app.seelf.environment"
)

type (
	Options interface {
		Domain() domain.Url
		AcmeEmail() string
	}

	Backend interface {
		domain.Backend
		Setup() error
	}

	DockerOptions func(*docker)

	docker struct {
		cli     command.Cli // Docker cli to use, if nil, a new one will be created per deployment task
		compose api.Service // Docker compose service to use, if nil, a new one will be created per deployment task
		options Options
		logger  log.Logger
	}
)

// Creates a docker backend with given options. The configuration is mostly used to
// ease the testing of some internals.
func New(options Options, logger log.Logger, configuration ...DockerOptions) Backend {
	d := &docker{
		options: options,
		logger:  logger,
	}

	for _, opt := range configuration {
		opt(d)
	}

	return d
}

// Use the given compose service and cli instead of creating new ones. Used for testing.
func WithDockerAndCompose(cli command.Cli, composeService api.Service) DockerOptions {
	return func(d *docker) {
		d.cli = cli
		d.compose = composeService
	}
}

func (d *docker) Setup() error {
	cli, compose, err := d.instantiateClientAndCompose(nil)

	if err != nil {
		return err
	}

	defer cli.Client().Close()

	d.logger.Info("deploying traefik balancer service, it could take a while if it's the first time...")

	project := &types.Project{
		Name: balancerProjectName,
		Services: []types.ServiceConfig{
			{
				Name:    balancerServiceName,
				Image:   "traefik:v2.6",
				Restart: types.RestartPolicyUnlessStopped,
				Command: []string{
					"--providers.docker",
					fmt.Sprintf("--providers.docker.network=%s", publicNetworkName),
					"--providers.docker.exposedbydefault=false",
				},
				Ports: []types.ServicePortConfig{
					{Target: 80, Published: "80"},
				},
				Volumes: []types.ServiceVolumeConfig{
					{Type: types.VolumeTypeBind, Source: "/var/run/docker.sock", Target: "/var/run/docker.sock"},
				},
				CustomLabels: getProjectCustomLabels(balancerProjectName, balancerServiceName, ""),
			},
		},
		Networks: types.Networks{
			"default": types.NetworkConfig{
				Name: publicNetworkName,
			},
		},
	}

	if d.options.Domain().UseSSL() {
		project.Services[0].Command = append(project.Services[0].Command,
			"--entrypoints.web.address=:80",
			"--entrypoints.web.http.redirections.entryPoint.to=websecure",
			"--entrypoints.web.http.redirections.entryPoint.scheme=https",
			"--entrypoints.websecure.address=:443",
			fmt.Sprintf("--certificatesresolvers.%s.acme.tlschallenge=true", certResolverName),
			fmt.Sprintf("--certificatesresolvers.%s.acme.email=%s", certResolverName, d.options.AcmeEmail()),
			fmt.Sprintf("--certificatesresolvers.%s.acme.storage=/letsencrypt/acme.json", certResolverName),
		)
		project.Services[0].Ports = append(project.Services[0].Ports, types.ServicePortConfig{
			Target: 443, Published: "443",
		})
		project.Services[0].Volumes = append(project.Services[0].Volumes, types.ServiceVolumeConfig{
			Type:   types.VolumeTypeVolume,
			Source: "letsencrypt",
			Target: "/letsencrypt",
		})

		project.Volumes = types.Volumes{
			"letsencrypt": types.VolumeConfig{},
		}
	}

	loader.Normalize(project)

	return compose.Up(context.Background(), project, api.UpOptions{
		Create: api.CreateOptions{
			RemoveOrphans: true,
			QuietPull:     true,
		},
		Start: api.StartOptions{
			Wait: true,
		},
	})
}

func (d *docker) Run(ctx context.Context, dir string, logger domain.DeploymentLogger, depl domain.Deployment) (domain.Services, error) {
	cli, compose, err := d.instantiateClientAndCompose(logger)

	if err != nil {
		return nil, err
	}

	defer cli.Client().Close()

	logger.Stepf("configuring seelf docker project for environment: %s", depl.Config().Environment())

	project, services, err := d.generateProject(depl, dir, logger)

	if err != nil {
		return nil, err
	}

	logger.Stepf("launching docker compose project (pulling, building and running)")

	if err = compose.Up(ctx, project, api.UpOptions{
		Create: api.CreateOptions{
			Build:         &api.BuildOptions{},
			RemoveOrphans: true,
		},
		Start: api.StartOptions{
			Wait: true,
		},
	}); err != nil {
		logger.Error(err)
		return nil, ErrComposeFailed
	}

	if d.options.Domain().UseSSL() {
		logger.Infof("you may have to wait for certificates to be generated before your app is available")
	}

	// Remove dangling images
	pruneResult, err := cli.Client().ImagesPrune(ctx, filters.NewArgs(
		filters.Arg("dangling", "true"),
		filters.Arg("label", fmt.Sprintf("%s=%s", AppLabel, depl.ID().AppID())),
		filters.Arg("label", fmt.Sprintf("%s=%s", EnvironmentLabel, depl.Config().Environment())),
	))

	if err == nil {
		prunedCount := len(pruneResult.ImagesDeleted)

		if prunedCount > 0 {
			logger.Infof("pruned %d dangling image(s)", prunedCount)
		}
	} else {
		// If there's an error, we just log it and go on since it's not a critical one
		logger.Warnf(err.Error())
	}

	return services, nil
}

func (d *docker) Cleanup(ctx context.Context, app domain.App) error {
	cli, _, err := d.instantiateClientAndCompose(nil)

	if err != nil {
		return err
	}

	client := cli.Client()

	defer client.Close()

	appFilters := filters.NewArgs(
		filters.Arg("label", fmt.Sprintf("%s=%s", AppLabel, app.ID())),
	)

	// List and stop all containers related to this application
	containers, err := client.ContainerList(ctx, dockertypes.ContainerListOptions{
		All:     true,
		Filters: appFilters,
	})

	if err != nil {
		return err
	}

	// Before removing containers, make sure everything is stopped
	for _, container := range containers {
		d.logger.Debugw("stopping container", "id", container.ID)
		if err = client.ContainerStop(ctx, container.ID, dockercontainer.StopOptions{}); err != nil {
			return err
		}
	}

	for _, container := range containers {
		d.logger.Debugw("removing container", "id", container.ID)
		if err = client.ContainerRemove(ctx, container.ID, dockertypes.ContainerRemoveOptions{}); err != nil {
			return err
		}
	}

	// List and remove all volumes
	volumes, err := client.VolumeList(ctx, volume.ListOptions{
		Filters: appFilters,
	})

	if err != nil {
		return err
	}

	for _, volume := range volumes.Volumes {
		d.logger.Debugw("removing volume", "name", volume.Name)
		if err = client.VolumeRemove(ctx, volume.Name, true); err != nil {
			return err
		}
	}

	// List and remove all networks
	networks, err := client.NetworkList(ctx, dockertypes.NetworkListOptions{
		Filters: appFilters,
	})

	if err != nil {
		return err
	}

	for _, network := range networks {
		d.logger.Debugw("removing network", "id", network.ID)
		if err = client.NetworkRemove(ctx, network.ID); err != nil {
			return err
		}
	}

	// List and remove all images
	images, err := client.ImageList(ctx, dockertypes.ImageListOptions{
		All:     true,
		Filters: appFilters,
	})

	if err != nil {
		return err
	}

	for _, image := range images {
		d.logger.Debugw("removing image", "id", image.ID)
		if _, err = client.ImageRemove(ctx, image.ID, dockertypes.ImageRemoveOptions{
			Force:         true,
			PruneChildren: true,
		}); err != nil {
			return err
		}
	}

	return nil
}

// Initialize a new docker client and compose service. You MUST close the command.Cli once done.
func (d *docker) instantiateClientAndCompose(logger domain.DeploymentLogger) (command.Cli, api.Service, error) {
	if d.compose != nil && d.cli != nil {
		return d.cli, d.compose, nil
	}

	stream := io.Discard

	if logger != nil {
		stream = logger
	}

	dockerCli, err := command.NewDockerCli(command.WithCombinedStreams(stream))

	if err != nil {
		return nil, nil, err
	}

	if err = dockerCli.Initialize(flags.NewClientOptions()); err != nil {
		return nil, nil, err
	}

	ping, err := dockerCli.Client().Ping(context.Background())

	if err != nil {
		return nil, nil, err
	}

	if logger != nil {
		logger.Stepf("connected to docker with version %s", ping.APIVersion)
	}

	return dockerCli, compose.NewComposeService(dockerCli), nil
}

// Generate a compose project for a specific app and transform it to make it usable
// by seelf (ie. exposing needed services)
//
// This function use some heuristics to determine what should be exposed and how
// according to the given configuration.
//
// The goal is really for the user to provide a docker-compose file which runs fine locally
// and we should do our best to expose it accordingly without the user providing anything.
func (d *docker) generateProject(depl domain.Deployment, dir string, logger domain.DeploymentLogger) (*types.Project, domain.Services, error) {
	var (
		services    domain.Services
		config      = depl.Config()
		seelfLabels = types.Labels{
			AppLabel:         string(depl.ID().AppID()),
			EnvironmentLabel: string(config.Environment()),
		}
	)

	composeFilepath, err := findServiceFile(dir, config.Environment())

	if err != nil {
		logger.Error(err)
		return nil, nil, ErrOpenComposeFileFailed
	}

	logger.Stepf("reading project from %s", composeFilepath)

	project, err := loadProject(composeFilepath, config.ProjectName(), config.Environment())

	if err != nil {
		logger.Error(err)
		return nil, nil, ErrLoadProjectFailed
	}

	disabledServicesCount := len(project.DisabledServices)

	if disabledServicesCount > 0 {
		disabledServicesNames := make([]string, disabledServicesCount)

		for i, service := range project.DisabledServices {
			disabledServicesNames[i] = service.Name
		}

		logger.Infof("some services have been disabled by the %s profile: %s", config.Environment(), strings.Join(disabledServicesNames, ", "))
	}

	// Sort services by alphabetical order so that we know how where the default subdomain (ie. the one without a service suffix)
	// will be.
	orderedNames := make([]string, len(project.Services))
	namesToIndex := make(map[string]int, len(project.Services))

	for i, service := range project.Services {
		orderedNames[i] = service.Name
		namesToIndex[service.Name] = i
	}

	slices.Sort(orderedNames)

	// Let's transform the project to expose needed services
	for _, name := range orderedNames {
		var (
			i               = namesToIndex[name]
			service         = project.Services[i]
			deployedService domain.Service
		)

		if len(service.Ports) == 0 {
			services, deployedService = services.Internal(config, service.Name, service.Image)
		} else {
			services, deployedService = services.Public(d.options.Domain(), config, service.Name, service.Image)
		}

		if service.Restart == "" {
			logger.Warnf("no restart policy sets for service %s, the service will not be restarted automatically", service.Name)
		}

		// If there's an image to build, force it (same as --build in the docker compose cli)
		if service.Build != nil {
			service.Image = deployedService.Image() // Since the image name may have been generated, override it
			service.PullPolicy = types.PullPolicyBuild
			service.Build.Labels = appendLabels(seelfLabels, service.Build.Labels)
		}

		// Attach environment variables if any
		servicesEnv := config.EnvironmentVariablesFor(deployedService.Name())

		if vars, hasVars := servicesEnv.TryGet(); hasVars {
			envNames := make([]string, 0, len(vars))

			for name, value := range vars {
				localValue := value // Copy the value to avoid the loop to use the same pointer
				service.Environment[name] = &localValue
				envNames = append(envNames, name)
			}

			logger.Infof("using %s environment variable(s) for service %s", strings.Join(envNames, ", "), deployedService.Name())
		}

		service.Labels = appendLabels(seelfLabels, service.Labels)

		for _, volume := range service.Volumes {
			if volume.Type == types.VolumeTypeBind {
				logger.Warnf("bind mount detected for service %s, this is not supported and your data are not guaranteed to be preserved, use docker volumes instead", deployedService.Name())
			}
		}

		// Not exposed, no need to go further
		if !deployedService.IsExposed() {
			project.Services[i] = service
			continue
		}

		url := deployedService.Url().MustGet()
		serviceQualifiedName := deployedService.QualifiedName()

		if len(service.Ports) > 1 {
			logger.Warnf("service %s exposes multiple ports but seelf only supports one port per service at the moment", deployedService.Name())
		}

		containerPort := uint64(service.Ports[0].Target)
		service.Ports = []types.ServicePortConfig{} // Remove them since traefik will expose this service

		if service.Networks == nil {
			service.Networks = map[string]*types.ServiceNetworkConfig{}
		}

		service.Networks[publicNetworkName] = nil // nil here because there's no additional options to give

		service.Labels["traefik.enable"] = "true"
		service.Labels[fmt.Sprintf("traefik.http.services.%s.loadbalancer.server.port", serviceQualifiedName)] = strconv.FormatUint(containerPort, 10)
		service.Labels[fmt.Sprintf("traefik.http.routers.%s.rule", serviceQualifiedName)] =
			fmt.Sprintf("Host(`%s`)", url.Host())

		if url.UseSSL() {
			service.Labels[fmt.Sprintf("traefik.http.routers.%s.tls.certresolver", serviceQualifiedName)] =
				certResolverName
		}

		project.Services[i] = service
	}

	// Add labels to network and volumes to make it easy to find them

	for name, network := range project.Networks {
		network.Labels = appendLabels(seelfLabels, network.Labels)
		project.Networks[name] = network
	}

	for name, volume := range project.Volumes {
		volume.Labels = appendLabels(seelfLabels, volume.Labels)
		project.Volumes[name] = volume
	}

	// Append the public seelf network to the project

	if project.Networks == nil {
		project.Networks = types.Networks{}
	}

	project.Networks[publicNetworkName] = types.NetworkConfig{
		Name: publicNetworkName,
		External: types.External{
			External: true,
		},
	}

	return project, services, nil
}

// add some labels to a given target.
func appendLabels(labelsToAdd types.Labels, target types.Labels) types.Labels {
	if target == nil {
		target = types.Labels{}
	}

	for k, v := range labelsToAdd {
		target[k] = v
	}

	return target
}

// Load a project from a given compose path.
func loadProject(composePath, projectName string, env domain.Environment) (*types.Project, error) {
	opts, err := cli.NewProjectOptions([]string{composePath},
		cli.WithName(projectName),
		cli.WithProfiles([]string{string(env)}),
	)

	if err != nil {
		return nil, err
	}

	project, err := cli.ProjectFromOptions(opts)

	if err != nil {
		return nil, err
	}

	for i, s := range project.Services {
		s.CustomLabels = getProjectCustomLabels(project.Name, s.Name, project.WorkingDir, project.ComposeFiles...)
		project.Services[i] = s
	}

	return project, nil
}

// Find the service file that should be used based on the given environment.
func findServiceFile(dir string, env domain.Environment) (string, error) {
	var (
		composeFilepath      string
		serviceFilesAffinity = []string{
			fmt.Sprintf("compose.seelf.%s.yml", env),
			fmt.Sprintf("compose.seelf.%s.yaml", env),
			fmt.Sprintf("docker-compose.seelf.%s.yml", env),
			fmt.Sprintf("docker-compose.seelf.%s.yaml", env),
			fmt.Sprintf("compose.%s.yml", env),
			fmt.Sprintf("compose.%s.yaml", env),
			fmt.Sprintf("docker-compose.%s.yml", env),
			fmt.Sprintf("docker-compose.%s.yaml", env),
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

	// Find the first valid docker-compose file
	for _, file := range serviceFilesAffinity {
		servicePath := filepath.Join(dir, file)
		_, err := os.Stat(servicePath)

		if os.IsNotExist(err) {
			continue
		}

		if err == nil {
			composeFilepath = servicePath
			break
		}

		return "", err
	}

	if composeFilepath == "" {
		return "", fmt.Errorf("could not find a valid compose file, tried in the following order:\n\t%s", strings.Join(serviceFilesAffinity, "\n\t"))
	}

	return composeFilepath, nil
}

// Apply common compose labels as per https://github.com/docker/compose/blob/126cb988c6f0c00a2a9887b8a39dc0907daec289/cmd/compose/compose.go#L200
func getProjectCustomLabels(project, service, workingDir string, composeFiles ...string) map[string]string {
	labels := map[string]string{
		api.ProjectLabel:     project,
		api.ServiceLabel:     service,
		api.VersionLabel:     api.ComposeVersion,
		api.ConfigFilesLabel: strings.Join(composeFiles, ","),
		api.OneoffLabel:      "False",
	}

	if workingDir != "" {
		labels[api.WorkingDirLabel] = workingDir
	}

	return labels
}
