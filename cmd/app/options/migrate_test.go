package options

import (
	"context"
	"testing"

	"github.com/tae2089/go-template/internal/config"
)

func TestMigrateCompletesValidatesAndRuns(t *testing.T) {
	var got config.Config
	migrate := NewMigrate(func(_ context.Context, cfg config.Config) error {
		got = cfg
		return nil
	})
	options := config.Options{
		Overrides: map[string]any{
			"serve.address": "invalid but irrelevant to migrate",
			"database.dsn":  "migration.db",
		},
	}

	if err := migrate.Complete(options); err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if err := migrate.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if err := migrate.Run(context.Background()); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got.Database.DSN != "migration.db" {
		t.Errorf("runner database DSN = %q, want %q", got.Database.DSN, "migration.db")
	}
}

func TestMigrateCompleteRejectsMissingRunner(t *testing.T) {
	migrate := NewMigrate(nil)

	err := migrate.Complete(config.Options{})

	if err == nil {
		t.Fatal("Complete() error = nil, want missing runner error")
	}
}
