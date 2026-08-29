package command

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/tae2089/go-template/internal/config"
)

func TestServeCommandRunsLifecycleInOrder(t *testing.T) {
	var events []string
	var got config.Options
	serveOptions := fakeServeOptions{
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

	cmd := NewServeCommand(config.Options{}, serveOptions)
	cmd.SetArgs([]string{"--address", ":9090"})

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext() error = %v", err)
	}
	if got, want := strings.Join(events, ","), "complete,validate,run"; got != want {
		t.Errorf("lifecycle order = %q, want %q", got, want)
	}
	address, err := got.FlagSet.GetString("address")
	if err != nil {
		t.Fatalf("get address flag: %v", err)
	}
	if address != ":9090" {
		t.Errorf("address flag = %q, want %q", address, ":9090")
	}
}

func TestServeCommandStopsAfterCompleteError(t *testing.T) {
	validateCalled := false
	runCalled := false
	serveOptions := fakeServeOptions{
		complete: func(config.Options) error {
			return errors.New("complete failed")
		},
		validate: func() error {
			validateCalled = true
			return nil
		},
		run: func(context.Context) error {
			runCalled = true
			return nil
		},
	}

	err := NewServeCommand(config.Options{}, serveOptions).ExecuteContext(context.Background())
	if err == nil || !strings.Contains(err.Error(), "complete failed") {
		t.Fatalf("ExecuteContext() error = %v, want complete error", err)
	}
	if validateCalled {
		t.Error("Validate() was called after Complete() failed")
	}
	if runCalled {
		t.Error("Run() was called after Complete() failed")
	}
}

func TestServeCommandStopsAfterValidateError(t *testing.T) {
	runCalled := false
	serveOptions := fakeServeOptions{
		complete: func(config.Options) error {
			return nil
		},
		validate: func() error {
			return errors.New("validate failed")
		},
		run: func(context.Context) error {
			runCalled = true
			return nil
		},
	}

	err := NewServeCommand(config.Options{}, serveOptions).ExecuteContext(context.Background())
	if err == nil || !strings.Contains(err.Error(), "validate failed") {
		t.Fatalf("ExecuteContext() error = %v, want validation error", err)
	}
	if runCalled {
		t.Error("Run() was called after Validate() failed")
	}
}

func TestServeCommandRejectsMissingOptions(t *testing.T) {
	err := NewServeCommand(config.Options{}, nil).ExecuteContext(context.Background())
	if err == nil || !strings.Contains(err.Error(), "serve options are required") {
		t.Fatalf("ExecuteContext() error = %v, want missing options error", err)
	}
}

type fakeServeOptions struct {
	complete func(config.Options) error
	validate func() error
	run      func(context.Context) error
}

func (o fakeServeOptions) Complete(opts config.Options) error {
	return o.complete(opts)
}

func (o fakeServeOptions) Validate() error {
	return o.validate()
}

func (o fakeServeOptions) Run(ctx context.Context) error {
	return o.run(ctx)
}
