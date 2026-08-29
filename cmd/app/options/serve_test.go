package options

import (
	"context"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/tae2089/go-template/internal/config"
)

func TestServeCompletesValidatesAndRuns(t *testing.T) {
	flags := newFlagSet(t, "--address", ":9090", "--database-dsn", "file:test.db")
	var got config.Config
	serve := NewServe(func(_ context.Context, cfg config.Config) error {
		got = cfg
		return nil
	})

	if err := serve.Complete(config.Options{FlagSet: flags}); err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if err := serve.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if err := serve.Run(context.Background()); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if got.Serve.Address != ":9090" {
		t.Errorf("runner address = %q, want %q", got.Serve.Address, ":9090")
	}
	if got.Database.DSN != "file:test.db" {
		t.Errorf("runner database DSN = %q, want %q", got.Database.DSN, "file:test.db")
	}
}

func TestServeValidateRejectsInvalidConfigurationBeforeRun(t *testing.T) {
	flags := newFlagSet(t, "--address", "invalid")
	called := false
	serve := NewServe(func(context.Context, config.Config) error {
		called = true
		return nil
	})

	if err := serve.Complete(config.Options{FlagSet: flags}); err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	err := serve.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want validation error")
	}
	if !strings.Contains(err.Error(), "validate configuration") {
		t.Errorf("Validate() error = %q, want validation context", err)
	}
	if called {
		t.Error("runner was called before validation succeeded")
	}
}

func TestServeCompleteRejectsMissingRunner(t *testing.T) {
	serve := NewServe(nil)

	err := serve.Complete(config.Options{FlagSet: newFlagSet(t)})
	if err == nil {
		t.Fatal("Complete() error = nil, want missing runner error")
	}
	if !strings.Contains(err.Error(), "serve runner is required") {
		t.Errorf("Complete() error = %q, want missing runner error", err)
	}
}

func newFlagSet(t *testing.T, args ...string) *pflag.FlagSet {
	t.Helper()

	flags := pflag.NewFlagSet("serve", pflag.ContinueOnError)
	flags.String("address", "", "")
	flags.String("database-driver", "", "")
	flags.String("database-dsn", "", "")
	if err := flags.Parse(args); err != nil {
		t.Fatalf("parse flags: %v", err)
	}
	return flags
}
