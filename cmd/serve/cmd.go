package serve

import (
	"github.com/YuukanOO/seelf/cmd/startup"
	"github.com/spf13/cobra"
)

type Options interface {
	ServerOptions
	startup.ServerOptions
}

// Returns the root serve command
func Root(opts Options) *cobra.Command {
	serveCmd := &cobra.Command{
		Use:   "serve",
		Short: "Launch the web application!",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := startup.Server(opts)

			if err != nil {
				return err
			}

			defer root.Cleanup()

			return newHttpServer(opts, root).Listen()
		},
	}

	return serveCmd
}
