package serve

import "github.com/spf13/cobra"

// Returns the root serve command
func Root(opts Options) *cobra.Command {
	serveCmd := &cobra.Command{
		Use:   "serve",
		Short: "Launch the web application!",
		RunE: func(cmd *cobra.Command, args []string) error {
			server, err := newHttpServer(opts)

			if err != nil {
				return err
			}

			defer server.Cleanup()

			return server.Listen()
		},
	}

	return serveCmd
}
