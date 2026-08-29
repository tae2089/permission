package command

import (
	"context"
	"testing"

	"github.com/tae2089/go-template/internal/config"
)

func TestRootCommandDispatchesToServe(t *testing.T) {
	var gotAddress string
	serveOptions := fakeServeOptions{
		complete: func(opts config.Options) error {
			address, err := opts.FlagSet.GetString("address")
			if err != nil {
				return err
			}
			gotAddress = address
			return nil
		},
		validate: func() error { return nil },
		run:      func(context.Context) error { return nil },
	}

	cmd := NewRootCommand("app", config.Options{}, serveOptions, nil)
	cmd.SetArgs([]string{"serve", "--address", ":9090"})

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext() error = %v", err)
	}
	if gotAddress != ":9090" {
		t.Errorf("address flag = %q, want %q", gotAddress, ":9090")
	}
}

func TestRootCommandUsesInjectedApplicationName(t *testing.T) {
	cmd := NewRootCommand("catalog", config.Options{}, fakeServeOptions{
		complete: func(config.Options) error { return nil },
		validate: func() error { return nil },
		run:      func(context.Context) error { return nil },
	}, nil)

	if cmd.Use != "catalog" {
		t.Errorf("command use = %q, want %q", cmd.Use, "catalog")
	}
	if cmd.Short != "Run catalog commands" {
		t.Errorf("command short = %q, want catalog command description", cmd.Short)
	}
}

func TestRootCommandDispatchesToMigrate(t *testing.T) {
	migrateCalled := false
	migrateOptions := fakeServeOptions{
		complete: func(config.Options) error { return nil },
		validate: func() error { return nil },
		run: func(context.Context) error {
			migrateCalled = true
			return nil
		},
	}
	cmd := NewRootCommand("app", config.Options{}, nil, migrateOptions)
	cmd.SetArgs([]string{"migrate"})

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext() error = %v", err)
	}
	if !migrateCalled {
		t.Error("migrate lifecycle was not run")
	}
}
