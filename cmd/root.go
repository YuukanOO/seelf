package cmd

import (
	"github.com/YuukanOO/seelf/cmd/config"
	"github.com/YuukanOO/seelf/cmd/serve"
	"github.com/YuukanOO/seelf/cmd/version"
	"github.com/spf13/cobra"
)

// Build the root command where everything start!
func Root() *cobra.Command {
	var (
		conf              = config.Default()
		configurationPath = conf.ConfigPath()
		isVerbose         = conf.IsVerbose()
	)

	rootCmd := &cobra.Command{
		Use:          "seelf",
		SilenceUsage: true,
		Version:      version.Current(),
		Short:        "Painless self-hosting in a single binary.",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return conf.Initialize(configurationPath, isVerbose)
		},
	}

	rootCmd.PersistentFlags().StringVarP(&configurationPath, "config", "c", configurationPath, "config file to use")
	rootCmd.PersistentFlags().BoolVarP(&isVerbose, "verbose", "v", isVerbose, "enable verbose mode")

	// Add sub-commands
	rootCmd.AddCommand(serve.Root(conf))

	return rootCmd
}
