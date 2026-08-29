package command

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/tae2089/go-template/internal/config"
)

func NewMigrateCommand(baseConfig config.Options, migrateOptions Lifecycle) *cobra.Command {
	configFile := baseConfig.ConfigFile

	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Apply database migrations",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if migrateOptions == nil {
				return errors.New("migrate options are required")
			}

			opts := baseConfig
			opts.ConfigFile = configFile
			opts.FlagSet = cmd.Flags()

			if err := migrateOptions.Complete(opts); err != nil {
				return err
			}
			if err := migrateOptions.Validate(); err != nil {
				return err
			}
			return migrateOptions.Run(cmd.Context())
		},
	}

	cmd.Flags().StringVar(&configFile, "config", configFile, "path to the configuration file")
	cmd.Flags().String("database-driver", "", "database driver")
	cmd.Flags().String("database-dsn", "", "SQLite database DSN")

	return cmd
}
