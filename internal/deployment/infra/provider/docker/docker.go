package docker

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/YuukanOO/seelf/internal/deployment/app/expose_seelf_container"
	"github.com/YuukanOO/seelf/internal/deployment/domain"
	"github.com/YuukanOO/seelf/internal/deployment/infra/provider"
	"github.com/YuukanOO/seelf/pkg/log"
	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/must"
	"github.com/YuukanOO/seelf/pkg/ssh"
	ptypes "github.com/YuukanOO/seelf/pkg/types"
	"github.com/YuukanOO/seelf/pkg/validate"
	"github.com/YuukanOO/seelf/pkg/validate/numbers"
	vstrings "github.com/YuukanOO/seelf/pkg/validate/strings"
	"github.com/compose-spec/compose-go/v2/cli"
	"github.com/compose-spec/compose-go/v2/types"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/flags"
	"github.com/docker/compose/v2/pkg/api"
	"github.com/docker/compose/v2/pkg/compose"
	dockertypes "github.com/docker/docker/api/types"
	dockercontainer "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/docker/docker/errdefs"
	"golang.org/x/exp/maps"
)

var (
	ErrLoadProjectFailed     = errors.New("compose_file_malformed")
	ErrOpenComposeFileFailed = errors.New("compose_file_open_failed")
	ErrComposeFailed         = errors.New("compose_failed")
	ErrTargetConnectFailed   = errors.New("target_connect_failed")

	sshConfigPath = filepath.Join(must.Panic(os.UserHomeDir()), ".ssh", "config")
)

const (
	AppLabel         = "app.seelf.application"
	TargetLabel      = "app.seelf.target"
	ExposedLabel     = "app.seelf.exposed"
	SubdomainLabel   = "app.seelf.subdomain"
	EnvironmentLabel = "app.seelf.environment"
)

type (
	DockerOptions func(*docker)

	Docker interface {
		provider.Provider
		expose_seelf_container.LocalProvider
	}

	docker struct {
		cli       command.Cli // Docker cli to use, if nil, a new one will be created per deployment task
		compose   api.Service // Docker compose service to use, if nil, a new one will be created per deployment task
		logger    log.Logger
		sshConfig ssh.Configurator
	}
)

// Creates a docker provider with given options. The configuration is mostly used to
// ease the testing of some internals.
//
// Multiple goroutine can use the same provider at the same time.
func New(logger log.Logger, configuration ...DockerOptions) Docker {
	d := &docker{
		logger:    logger,
		sshConfig: ssh.NewFileConfigurator(sshConfigPath),
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

func (d *docker) CanPrepare(payload any) bool                 { return ptypes.Is[Body](payload) }
func (d *docker) CanHandle(config domain.ProviderConfig) bool { return ptypes.Is[Data](config) }

func (d *docker) Prepare(ctx context.Context, payload any, existing ...domain.ProviderConfig) (domain.ProviderConfig, error) {
	config, ok := payload.(Body)

	if !ok {
		return nil, domain.ErrInvalidProviderPayload
	}

	var (
		host    ssh.Host
		privKey ssh.PrivateKey
	)

	if err := validate.Struct(validate.Of{
		"docker.host": validate.Maybe(config.Host, func(s string) error {
			return validate.Value(s, &host, ssh.ParseHost)
		}),
		"docker.user": validate.Maybe(config.User, vstrings.Required),
		"docker.port": validate.Maybe(config.Port, numbers.Min(0)),
		"docker.private_key": validate.Patch(config.PrivateKey, func(s string) error {
			return validate.Value(s, &privKey, ssh.ParsePrivateKey)
		}),
	}); err != nil {
		return nil, err
	}

	var data Data

	// No host, we're done
	if !config.Host.HasValue() {
		return data, nil
	}

	data.Host.Set(host)
	data.User.Set(config.User.Get(defaultUser))
	data.Port.Set(config.Port.Get(defaultPort))

	// Private key nil, we're done
	if config.PrivateKey.IsNil() {
		return data, nil
	}

	// Private key set, we're done
	if config.PrivateKey.HasValue() {
		data.PrivateKey.Set(privKey)
		return data, nil
	}

	// Else try to retrieve it from the existing one.
	// (This is needed because the existing private key is never exposed to the end user)
	for _, existingConfig := range existing {
		d, isDockerData := existingConfig.(Data)

		if !isDockerData {
			continue
		}

		data.PrivateKey = d.PrivateKey
	}

	return data, nil
}

func (d *docker) Setup(ctx context.Context, target domain.Target) error {
	config, ok := target.Provider().(Data)

	if !ok {
		return domain.ErrInvalidProviderPayload
	}

	id := target.ID()

	// If the target is a remote one, configure the appropriate ssh configuration to make
	// sure it's reachable.
	host, isRemote := config.Host.TryGet()

	if isRemote {
		var key monad.Maybe[ssh.ConnectionKey]

		if privKey, hasKey := config.PrivateKey.TryGet(); hasKey {
			key.Set(ssh.ConnectionKey{
				Name: string(id),
				Key:  privKey,
			})
		}

		if err := d.sshConfig.Upsert(ssh.Connection{
			Identifier: string(id),
			Host:       host,
			User:       config.User,
			Port:       config.Port,
			PrivateKey: key,
		}); err != nil {
			return err
		}
	}

	cli, compose, err := d.tryConnect(ctx, nil, config.Host)

	if err != nil {
		return err
	}

	defer cli.Client().Close()

	var (
		targetIdLower = strings.ToLower(string(id))
		projectName   = "seelf-internal-" + targetIdLower
		networkName   = targetPublicNetworkName(id)
	)

	// Append seelf specific labels to be able to clean up resources if the target is deleted
	labels := types.Labels{
		TargetLabel: string(id),
	}

	constraintsLabel := "--providers.docker.constraints=(Label(`" + TargetLabel + "`, `" + string(id) + "`) && LabelRegex(`" + SubdomainLabel + "`, `.+`))"

	// If it's a local docker, append a specific alternative constraint to allow this target to expose
	// seelf itself and make hosting it a breeze.
	if !isRemote {
		constraintsLabel += " || Label(`" + ExposedLabel + "`, `true`)"
	}

	project := &types.Project{
		Name: projectName,
		Services: types.Services{
			"proxy": types.ServiceConfig{
				Name:    "proxy",
				Labels:  labels,
				Image:   "traefik:v2.11",
				Restart: types.RestartPolicyUnlessStopped,
				Command: types.ShellCommand{
					"--providers.docker",
					"--providers.docker.network=" + networkName,
					constraintsLabel,
					"--providers.docker.defaultrule=Host(`{{ index .Labels " + `"` + SubdomainLabel + `"` + "}}." + target.Url().Host() + "`)",
				},
				Ports: []types.ServicePortConfig{
					{Target: 80, Published: "80"},
				},
				Volumes: []types.ServiceVolumeConfig{
					{Type: types.VolumeTypeBind, Source: "/var/run/docker.sock", Target: "/var/run/docker.sock"},
				},
				CustomLabels: getProjectCustomLabels(projectName, "proxy", ""),
			},
		},
		Networks: types.Networks{
			"default": types.NetworkConfig{
				Name:   networkName,
				Labels: labels,
			},
		},
	}

	if target.Url().UseSSL() {
		certResolverName := "seelf-resolver-" + targetIdLower

		proxyService := project.Services["proxy"]
		proxyService.Command = append(project.Services["proxy"].Command,
			"--entrypoints.web.address=:80",
			"--entrypoints.web.http.redirections.entryPoint.to=websecure",
			"--entrypoints.web.http.redirections.entryPoint.scheme=https",
			"--entrypoints.websecure.address=:443",
			"--certificatesresolvers."+certResolverName+".acme.tlschallenge=true",
			"--certificatesresolvers."+certResolverName+".acme.storage=/letsencrypt/acme.json",
			"--entrypoints.websecure.http.tls.certresolver="+certResolverName,
		)
		proxyService.Ports = append(proxyService.Ports, types.ServicePortConfig{
			Target: 443, Published: "443",
		})
		proxyService.Volumes = append(proxyService.Volumes, types.ServiceVolumeConfig{
			Type:   types.VolumeTypeVolume,
			Source: "letsencrypt",
			Target: "/letsencrypt",
		})
		project.Services["proxy"] = proxyService

		project.Volumes = types.Volumes{
			"letsencrypt": types.VolumeConfig{
				Name:   projectName + "_letsencrypt",
				Labels: labels,
			},
		}
	}

	return compose.Up(ctx, project, api.UpOptions{
		Create: api.CreateOptions{
			RemoveOrphans: true,
			QuietPull:     true,
		},
		Start: api.StartOptions{
			Wait: true,
		},
	})
}

func (d *docker) RemoveConfiguration(_ context.Context, target domain.Target) error {
	return d.sshConfig.Remove(string(target.ID()))
}

func (d *docker) PrepareLocal(context.Context) (domain.ProviderConfig, error) {
	return Data{}, nil
}

func (d *docker) Expose(ctx context.Context, target domain.Target, container string) error {
	cli, _, err := d.connect(ctx, nil, target)

	if err != nil {
		return err
	}

	client := cli.Client()

	defer client.Close()

	err = client.NetworkConnect(ctx, targetPublicNetworkName(target.ID()), container, nil)

	// Catch the forbidden error when the container is already attached
	if errdefs.IsForbidden(err) {
		return nil
	}

	if err != nil {
		return err
	}

	// Without the restart, traefik may pick the wrong container ip... This will force seelf to restart right now
	return client.ContainerRestart(ctx, container, dockercontainer.StopOptions{})
}

func (d *docker) Deploy(ctx context.Context, deploymentCtx domain.DeploymentContext, depl domain.Deployment, target domain.Target) (domain.Services, error) {
	logger := deploymentCtx.Logger()
	cli, compose, err := d.connect(ctx, logger, target)

	if err != nil {
		logger.Error(err)
		return nil, ErrTargetConnectFailed
	}

	client := cli.Client()

	defer client.Close()

	logger.Stepf("configuring seelf docker project for environment: %s", depl.Config().Environment())

	project, services, err := generateProject(ctx, depl, deploymentCtx.BuildDirectory(), logger)

	if err != nil {
		return nil, err
	}

	logger.Stepf("launching docker compose project (pulling, building and running)")

	if err = compose.Up(ctx, project, api.UpOptions{
		Create: api.CreateOptions{
			Build: &api.BuildOptions{
				Quiet: true,
			},
			RemoveOrphans: true,
		},
		Start: api.StartOptions{
			Wait: true,
		},
	}); err != nil {
		logger.Error(err)
		return nil, ErrComposeFailed
	}

	if target.Url().UseSSL() {
		logger.Infof("you may have to wait for certificates to be generated before your app is available")
	}

	// Remove dangling images
	pruneResult, err := client.ImagesPrune(ctx, filters.NewArgs(
		filters.Arg("dangling", "true"),
		filters.Arg("label", AppLabel+"="+string(depl.ID().AppID())),
		filters.Arg("label", TargetLabel+"="+string(target.ID())),
		filters.Arg("label", EnvironmentLabel+"="+string(depl.Config().Environment())),
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

func (d *docker) CleanupTarget(ctx context.Context, target domain.Target, strategy domain.CleanupStrategy) (err error) {
	if strategy == domain.CleanupStrategySkip {
		return nil
	}

	cli, _, err := d.connect(ctx, nil, target)

	if err != nil {
		return err
	}

	client := cli.Client()

	defer client.Close()

	// Remove all resources managed by seelf

	// TODO: We should probably prune all images and docker builder cache to free up some space

	return d.removeResources(ctx, client, filters.NewArgs(
		filters.Arg("label", TargetLabel+"="+string(target.ID())),
	))
}

func (d *docker) Cleanup(ctx context.Context, app domain.AppID, target domain.Target, env domain.Environment, strategy domain.CleanupStrategy) error {
	if strategy == domain.CleanupStrategySkip {
		return nil
	}

	cli, _, err := d.connect(ctx, nil, target)

	if err != nil {
		return err
	}

	client := cli.Client()

	defer client.Close()

	return d.removeResources(ctx, client, filters.NewArgs(
		filters.Arg("label", AppLabel+"="+string(app)),
		filters.Arg("label", TargetLabel+"="+string(target.ID())),
		filters.Arg("label", EnvironmentLabel+"="+string(env)),
	))
}

func (d *docker) tryConnect(ctx context.Context, logger domain.DeploymentLogger, host monad.Maybe[ssh.Host]) (command.Cli, api.Service, error) {
	// For tests, bypass the initialization and use the provided ones
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

	opts := flags.NewClientOptions()

	if h, isRemote := host.TryGet(); isRemote {
		opts.Hosts = append(opts.Hosts, "ssh://"+h.String())
	}

	if err = dockerCli.Initialize(opts); err != nil {
		return nil, nil, err
	}

	// Make sure the client is closed if an error occurs
	defer func(cli command.Cli) {
		if err != nil && cli != nil {
			cli.Client().Close()
		}
	}(dockerCli)

	ping, err := dockerCli.Client().Ping(ctx)

	if err != nil {
		return nil, nil, err
	}

	if logger != nil {
		logger.Stepf("successfully connected to docker version %s", ping.APIVersion)
	}

	return dockerCli, compose.NewComposeService(dockerCli), nil
}

// Connect to the docker daemon and return a new docker cli and compose service.
func (d *docker) connect(ctx context.Context, logger domain.DeploymentLogger, target domain.Target) (command.Cli, api.Service, error) {
	data, ok := target.Provider().(Data)

	if !ok {
		return nil, nil, domain.ErrInvalidProviderPayload
	}

	return d.tryConnect(ctx, logger, data.Host)
}

// Remove all resources matching the given filters
func (d *docker) removeResources(ctx context.Context, client client.APIClient, criterias filters.Args) error {
	// List and stop all containers related to this application
	containers, err := client.ContainerList(ctx, dockercontainer.ListOptions{
		All:     true,
		Filters: criterias,
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
		if err = client.ContainerRemove(ctx, container.ID, dockercontainer.RemoveOptions{}); err != nil {
			return err
		}
	}

	// List and remove all volumes
	volumes, err := client.VolumeList(ctx, volume.ListOptions{
		Filters: criterias,
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
		Filters: criterias,
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
		Filters: criterias,
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

// add some labels to a given target.
func appendLabels(target types.Labels, labelsToAdd types.Labels) types.Labels {
	if target == nil {
		target = types.Labels{}
	}

	for k, v := range labelsToAdd {
		target[k] = v
	}

	return target
}

// Load a project from a given compose path.
func loadProject(ctx context.Context, composePath, projectName string, env domain.Environment) (*types.Project, error) {
	opts, err := cli.NewProjectOptions([]string{composePath},
		cli.WithName(projectName),
		cli.WithNormalization(true),
		cli.WithProfiles([]string{string(env)}),
	)

	if err != nil {
		return nil, err
	}

	project, err := cli.ProjectFromOptions(ctx, opts)

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
		envStr               = string(env)
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

// Generate a compose project for a specific app and transform it to make it usable
// by seelf (ie. exposing needed services)
//
// This function use some heuristics to determine what should be exposed and how
// according to the given configuration.
//
// The goal is really for the user to provide a docker-compose file which runs fine locally
// and we should do our best to expose it accordingly without the user providing anything.
func generateProject(ctx context.Context, depl domain.Deployment, dir string, logger domain.DeploymentLogger) (*types.Project, domain.Services, error) {
	var (
		services    domain.Services
		config      = depl.Config()
		seelfLabels = types.Labels{
			AppLabel:         string(depl.ID().AppID()),
			TargetLabel:      string(config.Target()),
			EnvironmentLabel: string(config.Environment()),
		}
	)

	composeFilepath, err := findServiceFile(dir, config.Environment())

	if err != nil {
		logger.Error(err)
		return nil, nil, ErrOpenComposeFileFailed
	}

	logger.Stepf("reading project from %s", composeFilepath)

	project, err := loadProject(ctx, composeFilepath, config.ProjectName(), config.Environment())

	if err != nil {
		logger.Error(err)
		return nil, nil, ErrLoadProjectFailed
	}

	if len(project.DisabledServices) > 0 {
		logger.Infof("some services have been disabled by the %s profile: %s", config.Environment(), strings.Join(maps.Keys(project.DisabledServices), ", "))
	}

	// Sort services by alphabetical order so that we know how where the default subdomain (ie. the one without a service suffix)
	// will be.
	orderedNames := maps.Keys(project.Services)
	slices.Sort(orderedNames)

	publicNetworkName := targetPublicNetworkName(config.Target())

	// Let's transform the project to expose needed services
	for _, name := range orderedNames {
		var (
			service         = project.Services[name]
			deployedService domain.Service
		)

		services, deployedService = services.Append(config, service.Name, service.Image, len(service.Ports) > 0)

		if service.Restart == "" {
			logger.Warnf("no restart policy sets for service %s, the service will not be restarted automatically", service.Name)
		}

		// If there's an image to build, force it (same as --build in the docker compose cli)
		if service.Build != nil {
			service.Image = deployedService.Image() // Since the image name may have been generated, override it
			service.PullPolicy = types.PullPolicyBuild
			service.Build.Labels = appendLabels(service.Build.Labels, seelfLabels)
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

		service.Labels = appendLabels(service.Labels, seelfLabels)

		for _, volume := range service.Volumes {
			if volume.Type == types.VolumeTypeBind {
				logger.Warnf("bind mount detected for service %s, this is not supported and your data are not guaranteed to be preserved, use docker volumes instead", deployedService.Name())
			}
		}

		// Not exposed, no need to go further
		subdomain, isExposed := deployedService.Subdomain().TryGet()

		if !isExposed {
			project.Services[name] = service
			continue
		}

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

		service.Labels[SubdomainLabel] = subdomain
		service.Labels["traefik.http.services."+serviceQualifiedName+".loadbalancer.server.port"] = strconv.FormatUint(containerPort, 10)

		project.Services[name] = service
	}

	// Add labels to network and volumes to make it easy to find them

	for name, network := range project.Networks {
		network.Labels = appendLabels(network.Labels, seelfLabels)
		project.Networks[name] = network
	}

	for name, volume := range project.Volumes {
		volume.Labels = appendLabels(volume.Labels, seelfLabels)
		project.Volumes[name] = volume
	}

	// Append the public seelf network to the project

	if project.Networks == nil {
		project.Networks = types.Networks{}
	}

	project.Networks[publicNetworkName] = types.NetworkConfig{
		Name:     publicNetworkName,
		External: true,
	}

	return project, services, nil
}

// Retrieve the network name of a specific target
func targetPublicNetworkName(id domain.TargetID) string {
	return "seelf-gateway-" + strings.ToLower(string(id))
}
