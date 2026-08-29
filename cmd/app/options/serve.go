package options

import (
	"context"
	"errors"
	"fmt"

	"github.com/tae2089/go-template/internal/config"
)

type Runner func(context.Context, config.Config) error

type Serve struct {
	runner         Runner
	resolvedConfig config.Config
}

func NewServe(runner Runner) *Serve {
	return &Serve{runner: runner}
}

func (o *Serve) Complete(opts config.Options) error {
	if o.runner == nil {
		return errors.New("serve runner is required")
	}

	cfg, err := config.Load(opts)
	if err != nil {
		return err
	}
	o.resolvedConfig = cfg
	return nil
}

func (o *Serve) Validate() error {
	if err := o.resolvedConfig.Validate(); err != nil {
		return fmt.Errorf("validate configuration: %w", err)
	}
	return nil
}

func (o *Serve) Run(ctx context.Context) error {
	return o.runner(ctx, o.resolvedConfig)
}
