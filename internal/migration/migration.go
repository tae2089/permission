package migration

import (
	"context"
	"errors"

	"github.com/tae2089/trace/v3"

	"github.com/tae2089/go-template/internal/apikey"
	"github.com/tae2089/go-template/internal/audit"
	"github.com/tae2089/go-template/internal/config"
	"github.com/tae2089/go-template/internal/database"
	"github.com/tae2089/go-template/internal/project"
	"github.com/tae2089/go-template/internal/user"
)

func Run(ctx context.Context, cfg config.Config) error {
	connection, err := database.Open(ctx, database.Options{
		Driver: database.Driver(cfg.Database.Driver),
		DSN:    cfg.Database.DSN,
	})
	if err != nil {
		return trace.Wrap(err, "open migration database")
	}

	migrateErr := user.Migrate(ctx, connection.DB())
	if migrateErr == nil {
		migrateErr = audit.Migrate(ctx, connection.DB())
	}
	if migrateErr == nil {
		migrateErr = project.Migrate(ctx, connection.DB())
	}
	if migrateErr == nil {
		migrateErr = apikey.Migrate(ctx, connection.DB())
	}
	if migrateErr != nil {
		migrateErr = trace.Wrap(migrateErr, "apply database migrations")
	}
	closeErr := connection.Close()
	if closeErr != nil {
		closeErr = trace.Wrap(closeErr, "close migration database")
	}
	return errors.Join(migrateErr, closeErr)
}
