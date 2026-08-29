package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/tae2089/go-template/cmd/app/options"
	"github.com/tae2089/go-template/internal/command"
	"github.com/tae2089/go-template/internal/config"
	"github.com/tae2089/go-template/internal/migration"
	"github.com/tae2089/go-template/internal/server"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	serveOptions := options.NewServe(func(ctx context.Context, cfg config.Config) error {
		return server.Run(ctx, cfg, logger)
	})
	migrateOptions := options.NewMigrate(migration.Run)
	root := command.NewRootCommand(server.ApplicationName, config.Options{}, serveOptions, migrateOptions)
	if err := root.ExecuteContext(ctx); err != nil {
		logger.ErrorContext(
			context.Background(),
			"command failed",
			slog.String("error", err.Error()),
			slog.String("trace", fmt.Sprintf("%+v", err)),
		)
		os.Exit(1)
	}
}
