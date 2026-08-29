package command

import (
	"github.com/spf13/cobra"

	"github.com/tae2089/go-template/internal/config"
)

func NewRootCommand(
	applicationName string,
	baseConfig config.Options,
	serveOptions Lifecycle,
	migrateOptions Lifecycle,
) *cobra.Command {
	cmd := &cobra.Command{
		Use:           applicationName,
		Short:         "Run " + applicationName + " commands",
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	cmd.AddCommand(
		NewServeCommand(baseConfig, serveOptions),
		NewMigrateCommand(baseConfig, migrateOptions),
	)
	return cmd
}
