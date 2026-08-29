package migration

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/tae2089/go-template/internal/config"
	"github.com/tae2089/go-template/internal/database"
)

func TestRunAppliesFeatureSchemas(t *testing.T) {
	ctx := context.Background()
	dsn := filepath.Join(t.TempDir(), "migration.db")
	cfg := config.Config{
		Database: config.Database{
			Driver: string(database.DriverSQLite),
			DSN:    dsn,
		},
	}

	if err := Run(ctx, cfg); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	connection, err := database.Open(ctx, database.Options{
		Driver: database.DriverSQLite,
		DSN:    dsn,
	})
	if err != nil {
		t.Fatalf("open migrated database: %v", err)
	}
	t.Cleanup(func() {
		if err := connection.Close(); err != nil {
			t.Errorf("close migrated database: %v", err)
		}
	})
	if !connection.DB().Migrator().HasTable("users") {
		t.Error("users table does not exist after migration")
	}
	if !connection.DB().Migrator().HasTable("projects") {
		t.Error("projects table does not exist after migration")
	}
	if !connection.DB().Migrator().HasTable("audit_events") {
		t.Error("audit_events table does not exist after migration")
	}
	if !connection.DB().Migrator().HasTable("api_keys") {
		t.Error("api_keys table does not exist after migration")
	}
}
