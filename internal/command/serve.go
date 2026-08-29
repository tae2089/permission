package command

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/tae2089/go-template/internal/config"
)

func NewServeCommand(baseConfig config.Options, serveOptions Lifecycle) *cobra.Command {
	configFile := baseConfig.ConfigFile

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Run the server",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if serveOptions == nil {
				return errors.New("serve options are required")
			}

			opts := baseConfig
			opts.ConfigFile = configFile
			opts.FlagSet = cmd.Flags()

			if err := serveOptions.Complete(opts); err != nil {
				return err
			}
			if err := serveOptions.Validate(); err != nil {
				return err
			}
			return serveOptions.Run(cmd.Context())
		},
	}

	cmd.Flags().StringVar(&configFile, "config", configFile, "path to the configuration file")
	cmd.Flags().String("address", "", "server listen address")
	cmd.Flags().String("database-driver", "", "database driver")
	cmd.Flags().String("database-dsn", "", "SQLite database DSN")

	return cmd
}
