package docker

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
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
	"github.com/compose-spec/compose-go/v2/types"
	"github.com/docker/cli/cli/command"
	"github.com/docker/compose/v2/pkg/api"
	dockercontainer "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/errdefs"
)

var (
	ErrLoadProjectFailed     = errors.New("compose_file_malformed")
	ErrOpenComposeFileFailed = errors.New("compose_file_open_failed")
	ErrComposeFailed         = errors.New("compose_failed")
	ErrTargetConnectFailed   = errors.New("target_connect_failed")

	sshConfigPath = filepath.Join(must.Panic(os.UserHomeDir()), ".ssh", "config")
)

const (
	AppLabel               = "app.seelf.application"        // ID of the application
	EnvironmentLabel       = "app.seelf.environment"        // Environment of the application
	TargetLabel            = "app.seelf.target"             // ID of the target
	ExposedLabel           = "app.seelf.exposed"            // Force the exposure of a service (used when exposing seelf itself for example)
	SubdomainLabel         = "app.seelf.subdomain"          // Subdomain to use for the service, only for http entrypoints
	CustomEntrypointsLabel = "app.seelf.custom_entrypoints" // Boolean representing wether or not a service use custom entrypoints
)

type (
	DockerOptions func(*docker)

	Docker interface {
		provider.Provider
		expose_seelf_container.LocalProvider
	}

	docker struct {
		client    *client // Client to use, mostly for testing
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
		d.client = &client{
			cli:     cli,
			api:     cli.Client(),
			compose: composeService,
		}
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

func (d *docker) Setup(ctx context.Context, target domain.Target) (domain.TargetEntrypointsAssigned, error) {
	config, ok := target.Provider().(Data)

	if !ok {
		return nil, domain.ErrInvalidProviderPayload
	}

	if err := d.configureTargetSSH(target.ID(), config); err != nil {
		return nil, err
	}

	client, err := d.tryConnect(ctx, nil, config.Host)

	if err != nil {
		return nil, err
	}

	defer client.Close()

	if target.IsManual() {
		return nil, client.compose.Down(ctx, targetProjectName(target.ID()), api.DownOptions{
			RemoveOrphans: true,
			Images:        "all",
			Volumes:       true,
		})
	}

	project, assigned, err := newProxyProjectBuilder(client, target).Build(ctx)

	if err != nil {
		return nil, err
	}

	return assigned, client.compose.Up(ctx, project, api.UpOptions{
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
	client, err := d.connect(ctx, nil, target)

	if err != nil {
		return err
	}

	defer client.Close()

	err = client.api.NetworkConnect(ctx, targetPublicNetworkName(target.ID()), container, nil)

	// Catch the forbidden error when the container is already attached
	if errdefs.IsForbidden(err) {
		return nil
	}

	if err != nil {
		return err
	}

	// Without the restart, traefik may pick the wrong container ip... This will force seelf to restart right now
	return client.api.ContainerRestart(ctx, container, dockercontainer.StopOptions{})
}

func (d *docker) Deploy(
	ctx context.Context,
	deploymentCtx domain.DeploymentContext,
	deployment domain.Deployment,
	target domain.Target,
	registries []domain.Registry,
) (domain.Services, error) {
	logger := deploymentCtx.Logger()
	client, err := d.connect(ctx, logger, target, registries...)

	if err != nil {
		logger.Error(err)
		return nil, ErrTargetConnectFailed
	}

	defer client.Close()

	logger.Stepf("successfully connected to docker version %s", client.version)

	if len(client.registries) > 0 {
		logger.Infof("using custom registries: %s", strings.Join(client.registries, ", "))
	}

	project, services, err := newDeploymentProjectBuilder(deploymentCtx, deployment, target.IsManual()).Build(ctx)

	if err != nil {
		return nil, err
	}

	logger.Stepf("launching docker compose project (pulling, building and running)")

	if err = client.compose.Up(ctx, project, api.UpOptions{
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

	if url, isManagedBySeelf := target.Url().TryGet(); isManagedBySeelf {
		if url.UseSSL() {
			logger.Infof("you may have to wait for certificates to be generated before your app is available")
		}

		if len(services.CustomEntrypoints()) > 0 {
			logger.Infof("this deployment uses custom entrypoints. If this is the first time, you may have to wait a few seconds for the target to find available ports and expose them appropriately")
		}
	}

	prunedCount, err := client.PruneImages(ctx, filters.NewArgs(
		filters.Arg("dangling", "true"),
		filters.Arg("label", AppLabel+"="+string(deployment.ID().AppID())),
		filters.Arg("label", TargetLabel+"="+string(target.ID())),
		filters.Arg("label", EnvironmentLabel+"="+string(deployment.Config().Environment())),
	))

	if err != nil {
		logger.Warnf(err.Error())
	} else if prunedCount > 0 {
		logger.Infof("pruned %d dangling image(s)", prunedCount)
	}

	return services, nil
}

func (d *docker) CleanupTarget(ctx context.Context, target domain.Target, strategy domain.CleanupStrategy) (err error) {
	if strategy == domain.CleanupStrategySkip {
		return nil
	}

	client, err := d.connect(ctx, nil, target)

	if err != nil {
		return err
	}

	defer client.Close()

	// TODO: We should probably prune all images and docker builder cache to free up some space
	return client.RemoveResources(ctx, filters.NewArgs(
		filters.Arg("label", TargetLabel+"="+string(target.ID())),
	))
}

func (d *docker) Cleanup(ctx context.Context, app domain.AppID, target domain.Target, env domain.Environment, strategy domain.CleanupStrategy) error {
	if strategy == domain.CleanupStrategySkip {
		return nil
	}

	client, err := d.connect(ctx, nil, target)

	if err != nil {
		return err
	}

	defer client.Close()

	return client.RemoveResources(ctx, filters.NewArgs(
		filters.Arg("label", AppLabel+"="+string(app)),
		filters.Arg("label", TargetLabel+"="+string(target.ID())),
		filters.Arg("label", EnvironmentLabel+"="+string(env)),
	))
}

func (d *docker) tryConnect(ctx context.Context, out io.Writer, host monad.Maybe[ssh.Host], registries ...domain.Registry) (*client, error) {
	// For tests, bypass the initialization and use the provided one
	if d.client != nil {
		return d.client, nil
	}

	return connect(ctx, out, host, registries...)
}

// Connect to the docker daemon and return a new docker cli and compose service.
func (d *docker) connect(ctx context.Context, logger domain.DeploymentLogger, target domain.Target, registries ...domain.Registry) (*client, error) {
	data, ok := target.Provider().(Data)

	if !ok {
		return nil, domain.ErrInvalidProviderPayload
	}

	return d.tryConnect(ctx, logger, data.Host, registries...)
}

func (d *docker) configureTargetSSH(id domain.TargetID, config Data) error {
	host, isRemote := config.Host.TryGet()

	if !isRemote {
		return nil
	}

	var key monad.Maybe[ssh.ConnectionKey]

	if privKey, hasKey := config.PrivateKey.TryGet(); hasKey {
		key.Set(ssh.ConnectionKey{
			Name: string(id),
			Key:  privKey,
		})
	}

	return d.sshConfig.Upsert(ssh.Connection{
		Identifier: string(id),
		Host:       host,
		User:       config.User,
		Port:       config.Port,
		PrivateKey: key,
	})
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
