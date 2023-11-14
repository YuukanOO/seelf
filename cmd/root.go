package cmd

import (
	"github.com/YuukanOO/seelf/cmd/serve"
	"github.com/YuukanOO/seelf/pkg/config"
	"github.com/spf13/cobra"
)

// // Build the root command where everything start!
func Root() *cobra.Command {
	conf := DefaultConfiguration()

	rootCmd := &cobra.Command{
		Use:          "seelf",
		SilenceUsage: true,
		Version:      conf.CurrentVersion(),
		Short:        "Painless self-hosting in a single binary.",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if err := config.Load(conf.path, conf); err != nil {
				return err
			}

			return config.Save(conf.path, conf)
		},
	}

	rootCmd.PersistentFlags().StringVarP(&conf.path, "config", "c", conf.path, "config file to use")
	rootCmd.PersistentFlags().BoolVarP(&conf.Verbose, "verbose", "v", conf.Verbose, "enable verbose mode")

	rootCmd.AddCommand(serve.Root(conf))

	return rootCmd
}
