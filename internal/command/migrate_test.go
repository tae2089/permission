package command

import (
	"context"
	"strings"
	"testing"

	"github.com/tae2089/go-template/internal/config"
)

func TestMigrateCommandRunsLifecycleWithDatabaseFlags(t *testing.T) {
	var events []string
	var got config.Options
	migrateOptions := fakeServeOptions{
		complete: func(opts config.Options) error {
			events = append(events, "complete")
			got = opts
			return nil
		},
		validate: func() error {
			events = append(events, "validate")
			return nil
		},
		run: func(context.Context) error {
			events = append(events, "run")
			return nil
		},
	}
	cmd := NewMigrateCommand(config.Options{}, migrateOptions)
	cmd.SetArgs([]string{"--database-dsn", "migration.db"})

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext() error = %v", err)
	}
	if got, want := strings.Join(events, ","), "complete,validate,run"; got != want {
		t.Errorf("lifecycle order = %q, want %q", got, want)
	}
	dsn, err := got.FlagSet.GetString("database-dsn")
	if err != nil {
		t.Fatalf("get database DSN flag: %v", err)
	}
	if dsn != "migration.db" {
		t.Errorf("database DSN flag = %q, want %q", dsn, "migration.db")
	}
	if got.FlagSet.Lookup("address") != nil {
		t.Error("migrate command includes unrelated address flag")
	}
}

func TestMigrateCommandRejectsMissingOptions(t *testing.T) {
	err := NewMigrateCommand(config.Options{}, nil).ExecuteContext(context.Background())
	if err == nil || !strings.Contains(err.Error(), "migrate options are required") {
		t.Fatalf("ExecuteContext() error = %v, want missing options error", err)
	}
}
