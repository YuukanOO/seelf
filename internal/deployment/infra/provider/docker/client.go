package docker

import (
	"context"
	"io"

	"github.com/YuukanOO/seelf/pkg/monad"
	"github.com/YuukanOO/seelf/pkg/ssh"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/flags"
	"github.com/docker/compose/v2/pkg/api"
	"github.com/docker/compose/v2/pkg/compose"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/volume"
	dclient "github.com/docker/docker/client"
)

// Wraps a docker API client and compose service and expose some utility methods.
type client struct {
	cli     command.Cli
	api     dclient.APIClient
	compose api.Service
	version string
}

func connect(ctx context.Context, out io.Writer, host monad.Maybe[ssh.Host]) (*client, error) {
	stream := io.Discard

	if out != nil {
		stream = out
	}

	dockerCli, err := command.NewDockerCli(command.WithCombinedStreams(stream))

	if err != nil {
		return nil, err
	}

	opts := flags.NewClientOptions()

	if h, isRemote := host.TryGet(); isRemote {
		opts.Hosts = append(opts.Hosts, "ssh://"+h.String())
	}

	if err = dockerCli.Initialize(opts); err != nil {
		return nil, err
	}

	ping, err := dockerCli.Client().Ping(ctx)

	if err != nil {
		dockerCli.Client().Close()
		return nil, err
	}

	return &client{
		cli:     dockerCli,
		api:     dockerCli.Client(),
		version: ping.APIVersion,
		compose: compose.NewComposeService(dockerCli),
	}, nil
}

func (c *client) PruneImages(ctx context.Context, criteria filters.Args) (int, error) {
	// Remove dangling images
	pruneResult, err := c.api.ImagesPrune(ctx, criteria)

	if err != nil {
		return 0, err
	}

	return len(pruneResult.ImagesDeleted), nil
}

// Remove all resources matching the given filters
func (c *client) RemoveResources(ctx context.Context, criteria filters.Args) error {
	// List and stop all containers related to this application
	containers, err := c.api.ContainerList(ctx, container.ListOptions{
		All:     true,
		Filters: criteria,
	})

	if err != nil {
		return err
	}

	// Before removing containers, make sure everything is stopped
	for _, cont := range containers {
		if err = c.api.ContainerStop(ctx, cont.ID, container.StopOptions{}); err != nil {
			return err
		}
	}

	for _, cont := range containers {
		if err = c.api.ContainerRemove(ctx, cont.ID, container.RemoveOptions{}); err != nil {
			return err
		}
	}

	// List and remove all volumes
	volumes, err := c.api.VolumeList(ctx, volume.ListOptions{
		Filters: criteria,
	})

	if err != nil {
		return err
	}

	for _, volume := range volumes.Volumes {
		if err = c.api.VolumeRemove(ctx, volume.Name, true); err != nil {
			return err
		}
	}

	// List and remove all networks
	networks, err := c.api.NetworkList(ctx, types.NetworkListOptions{
		Filters: criteria,
	})

	if err != nil {
		return err
	}

	for _, network := range networks {
		if err = c.api.NetworkRemove(ctx, network.ID); err != nil {
			return err
		}
	}

	// List and remove all images
	images, err := c.api.ImageList(ctx, image.ListOptions{
		All:     true,
		Filters: criteria,
	})

	if err != nil {
		return err
	}

	for _, img := range images {
		if _, err = c.api.ImageRemove(ctx, img.ID, image.RemoveOptions{
			Force:         true,
			PruneChildren: true,
		}); err != nil {
			return err
		}
	}

	return nil
}

func (c *client) close() error {
	return c.api.Close()
}
