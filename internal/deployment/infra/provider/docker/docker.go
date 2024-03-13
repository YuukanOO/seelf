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
	"sync"

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
	"github.com/docker/docker/client"
)

var (
	ErrLoadProjectFailed         = errors.New("compose_file_malformed")
	ErrOpenComposeFileFailed     = errors.New("compose_file_open_failed")
	ErrComposeFailed             = errors.New("compose_failed")
	ErrTargetInitFailed          = errors.New("target_init_failed")
	ErrTargetNotAvailableAnymore = errors.New("target_not_available_anymore")

	sshConfigPath = filepath.Join(must.Panic(os.UserHomeDir()), ".ssh", "config")
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
		// FIXME: Add option to self expose seelf maybe
	}

	DockerOptions func(*docker)

	docker struct {
		cli                command.Cli // Docker cli to use, if nil, a new one will be created per deployment task
		compose            api.Service // Docker compose service to use, if nil, a new one will be created per deployment task
		options            Options
		logger             log.Logger
		sshConfig          ssh.Configurator
		targetsInitialized map[domain.TargetID]bool // Maps of target already initialized
		staleTargets       map[domain.TargetID]bool // Maps of target to consider as stale (ie. they will need a new setup before being used again)
		deletedTargets     map[domain.TargetID]bool // Maps of target to consider as deleted (not available anymore)
		muSetup            sync.Mutex               // Mutex to protect the targetsInitialized map
		muStaleAndDeleted  sync.Mutex               // Mutex to protect the staleTargets and deletedTargets map
	}
)

// Creates a docker provider with given options. The configuration is mostly used to
// ease the testing of some internals.
//
// Multiple goroutine can use the same provider at the same time.
func New(options Options, logger log.Logger, configuration ...DockerOptions) provider.Provider {
	d := &docker{
		options:            options,
		logger:             logger,
		sshConfig:          ssh.NewFileConfigurator(sshConfigPath),
		targetsInitialized: make(map[domain.TargetID]bool),
		staleTargets:       make(map[domain.TargetID]bool),
		deletedTargets:     make(map[domain.TargetID]bool),
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

func (d *docker) Run(ctx context.Context, deploymentCtx domain.DeploymentContext, depl domain.Deployment, target domain.Target) (domain.Services, error) {
	logger := deploymentCtx.Logger()
	cli, compose, err := d.setup(ctx, logger, target)

	if err != nil {
		return nil, err
	}

	client := cli.Client()

	defer client.Close()

	logger.Stepf("configuring seelf docker project for environment: %s", depl.Config().Environment())

	domain := target.Domain()

	project, services, err := generateProject(depl, deploymentCtx.BuildDirectory(), logger, domain)

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

	if domain.UseSSL() {
		logger.Infof("you may have to wait for certificates to be generated before your app is available")
	}

	// Remove dangling images
	pruneResult, err := client.ImagesPrune(ctx, filters.NewArgs(
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

func (d *docker) Stale(ctx context.Context, id domain.TargetID) error {
	d.muStaleAndDeleted.Lock()
	defer d.muStaleAndDeleted.Unlock()

	// Already in deleted targets, no need to mark it as stale
	if d.deletedTargets[id] {
		return nil
	}

	d.staleTargets[id] = true

	return nil
}

func (d *docker) CleanupTarget(ctx context.Context, target domain.Target) error {
	cli, _, err := d.setup(ctx, nil, target)

	if err != nil {
		// FIXME: Handle this case gracefully. One possibility is to return the err only if at least one
		// deployment has occured on the target. If an error is returned, the scheduled job will never succeed,
		// and so for this to be fixed, we need to expose scheduled jobs on the UI and the ability
		// to manually remove them, not the priority right now.
		d.logger.Errorw("failed to connect to target for cleanup, you may have to remove seelf resources yourself for now (you can use docker <resource> --filters label=app.seelf.application)",
			"target", target.ID(),
			"config", target.Provider().String(),
			"error", err)

		return d.removeTargetConfiguration(target.ID(), target.Provider().(Data))
	}

	client := cli.Client()

	defer client.Close()

	// Remove all resources managed by seelf
	if err = d.removeResources(ctx, client, filters.NewArgs(
		filters.Arg("label", AppLabel),
	)); err != nil {
		return err
	}

	return d.removeTargetConfiguration(target.ID(), target.Provider().(Data))
}

func (d *docker) Cleanup(ctx context.Context, app domain.AppID, target domain.Target, env domain.Environment) error {
	cli, _, err := d.setup(ctx, nil, target)

	if err != nil {
		return err
	}

	client := cli.Client()

	defer client.Close()

	return d.removeResources(ctx, client, filters.NewArgs(
		filters.Arg("label", fmt.Sprintf("%s=%s", AppLabel, app)),
		filters.Arg("label", fmt.Sprintf("%s=%s", EnvironmentLabel, env)),
	))
}

// Initialize a new docker client and compose service. You MUST close the command.Cli
// once done if no error is returned. The DeploymentLogger is optional.
//
// If that's the first time the target is used, it will make sure it is correctly
// initialized to actually expose services.
//
// This method is a bit hard to understand but its goal is to prevent multiple targets
// to be initialized at the same time which may cause issues.
//
// It handles rare cases where a deleted target could be requested and remove stale ones so
// that it could be correctly reintialized the next time its seen.
func (d *docker) setup(ctx context.Context, logger domain.DeploymentLogger, target domain.Target) (cli command.Cli, compose api.Service, err error) {
	id := target.ID()
	config, ok := target.Provider().(Data)

	if !ok {
		return nil, nil, domain.ErrInvalidProviderPayload
	}

	if logger != nil {
		logger.Stepf("checking if the target has already been initialized")
	}

	// Prevent multiple target initialization at the same time
	d.muSetup.Lock()
	defer d.muSetup.Unlock()

	if err := d.checkStaleAndDeletedTarget(d.targetsInitialized, id); err != nil {
		return nil, nil, err
	}

	var ping dockertypes.Ping

	// Translate the  error returned to a more generic one but logs the internal to make
	// sure the user knows what's going on.
	defer func() {
		if err == nil {
			if logger != nil {
				logger.Stepf("successfully connected to docker version %s", ping.APIVersion)
			}
			return
		}

		if logger != nil {
			logger.Error(err)
		} else {
			d.logger.Error(err)
		}

		err = ErrTargetInitFailed

		// If the client has been opened, close it
		if cli != nil {
			cli.Client().Close()
		}
	}()

	// Target already initialized, just try to connect to it
	if d.targetsInitialized[id] {
		if logger != nil {
			logger.Stepf("target already initialized during seelf lifetime, skipping setup")
		}

		cli, compose, ping, err = d.connect(ctx, logger, config.Host)

		return
	}

	if logger != nil {
		logger.Stepf("target is new or has changed, initializing...")
	}

	if cli, compose, ping, err = d.configureTarget(ctx, logger, id, target.Domain().UseSSL(), config); err != nil {
		return
	}

	d.targetsInitialized[id] = true

	return
}

// Connect to the docker daemon and return a new docker cli and compose service.
func (d *docker) connect(
	ctx context.Context,
	out io.Writer,
	host monad.Maybe[ssh.Host],
) (command.Cli, api.Service, dockertypes.Ping, error) {
	var ping dockertypes.Ping

	// For tests, bypass the initialization and use the provided ones
	if d.compose != nil && d.cli != nil {
		return d.cli, d.compose, ping, nil
	}

	stream := io.Discard

	if out != nil {
		stream = out
	}

	dockerCli, err := command.NewDockerCli(command.WithCombinedStreams(stream))

	if err != nil {
		return nil, nil, ping, err
	}

	opts := flags.NewClientOptions()

	if h, isRemote := host.TryGet(); isRemote {
		opts.Hosts = append(opts.Hosts, "ssh://"+h.String())
	}

	if err = dockerCli.Initialize(opts); err != nil {
		return nil, nil, ping, err
	}

	if ping, err = dockerCli.Client().Ping(ctx); err != nil {
		return nil, nil, ping, err
	}

	return dockerCli, compose.NewComposeService(dockerCli), ping, nil
}

// Mark the given target as deleted and remove the ssh configuration associated if needed.
func (d *docker) removeTargetConfiguration(id domain.TargetID, config Data) error {
	d.muStaleAndDeleted.Lock()
	defer d.muStaleAndDeleted.Unlock()

	if host, isRemote := config.Host.TryGet(); isRemote {
		if err := d.sshConfig.Remove(host, string(id)); err != nil {
			return err
		}
	}

	d.deletedTargets[id] = true

	return nil
}

// Remove all resources matching the given filters
func (d *docker) removeResources(ctx context.Context, client client.APIClient, filters filters.Args) error {
	// List and stop all containers related to this application
	containers, err := client.ContainerList(ctx, dockertypes.ContainerListOptions{
		All:     true,
		Filters: filters,
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
		Filters: filters,
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
		Filters: filters,
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
		Filters: filters,
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

// Check stale and deleted targets and remove them from the given map.
// If the given id appears to have been deleted, it will return an error.
func (d *docker) checkStaleAndDeletedTarget(targets map[domain.TargetID]bool, id domain.TargetID) error {
	d.muStaleAndDeleted.Lock()
	defer d.muStaleAndDeleted.Unlock()

	for deletedId := range d.deletedTargets {
		if deletedId == id {
			return ErrTargetNotAvailableAnymore
		}

		delete(d.targetsInitialized, deletedId)
	}

	for staleId := range d.staleTargets {
		delete(targets, staleId)
	}

	clear(d.deletedTargets)
	clear(d.staleTargets)

	return nil
}

// This method makes sure the target is reachable (by writing appropriate ssh configuration
// if needed) and deploy the proxy needed to expose services on it.
func (d *docker) configureTarget(
	ctx context.Context,
	logger domain.DeploymentLogger,
	id domain.TargetID,
	useSSL bool,
	config Data,
) (cli command.Cli, compose api.Service, ping dockertypes.Ping, err error) {
	// If the target is a remote one, configure the appropriate ssh configuration to make
	// sure it's reachable.
	if host, isRemote := config.Host.TryGet(); isRemote {
		var key monad.Maybe[ssh.ConnectionKey]

		if privKey, hasKey := config.PrivateKey.TryGet(); hasKey {
			key.Set(ssh.ConnectionKey{
				Name: string(id),
				Key:  privKey,
			})
		}

		if err = d.sshConfig.Upsert(ssh.Connection{
			Identifier: string(id),
			Host:       host,
			User:       config.User,
			Port:       config.Port,
			PrivateKey: key,
		}); err != nil {
			return
		}
	}

	if cli, compose, ping, err = d.connect(ctx, logger, config.Host); err != nil {
		return
	}

	// Deploy the traefik proxy
	if logger != nil {
		logger.Stepf("deploying proxy service, it could take a while if it's the first time")
	}

	// Append seelf specific labels to be able to clean up resources if the target is deleted
	labels := types.Labels{AppLabel: balancerServiceName}

	project := &types.Project{
		Name: balancerProjectName,
		Services: []types.ServiceConfig{
			{
				Name:    balancerServiceName,
				Labels:  labels,
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
				Name:   publicNetworkName,
				Labels: labels,
			},
		},
	}

	if useSSL {
		project.Services[0].Command = append(project.Services[0].Command,
			"--entrypoints.web.address=:80",
			"--entrypoints.web.http.redirections.entryPoint.to=websecure",
			"--entrypoints.web.http.redirections.entryPoint.scheme=https",
			"--entrypoints.websecure.address=:443",
			fmt.Sprintf("--certificatesresolvers.%s.acme.tlschallenge=true", certResolverName),
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
			"letsencrypt": types.VolumeConfig{
				Labels: labels,
			},
		}
	}

	if err = loader.Normalize(project); err != nil {
		return
	}

	err = compose.Up(ctx, project, api.UpOptions{
		Create: api.CreateOptions{
			RemoveOrphans: true,
			QuietPull:     true,
		},
		Start: api.StartOptions{
			Wait: true,
		},
	})

	return
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

// Generate a compose project for a specific app and transform it to make it usable
// by seelf (ie. exposing needed services)
//
// This function use some heuristics to determine what should be exposed and how
// according to the given configuration.
//
// The goal is really for the user to provide a docker-compose file which runs fine locally
// and we should do our best to expose it accordingly without the user providing anything.
func generateProject(depl domain.Deployment, dir string, logger domain.DeploymentLogger, targetUrl domain.Url) (*types.Project, domain.Services, error) {
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
			services, deployedService = services.Public(targetUrl, config, service.Name, service.Image)
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
