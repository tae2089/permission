package options

import (
	"context"
	"errors"
	"fmt"

	"github.com/tae2089/go-template/internal/config"
)

type Migrate struct {
	runner         Runner
	resolvedConfig config.Config
}

func NewMigrate(runner Runner) *Migrate {
	return &Migrate{runner: runner}
}

func (o *Migrate) Complete(opts config.Options) error {
	if o.runner == nil {
		return errors.New("migrate runner is required")
	}

	cfg, err := config.Load(opts)
	if err != nil {
		return err
	}
	o.resolvedConfig = cfg
	return nil
}

func (o *Migrate) Validate() error {
	if err := o.resolvedConfig.Database.Validate(); err != nil {
		return fmt.Errorf("validate database configuration: %w", err)
	}
	return nil
}

func (o *Migrate) Run(ctx context.Context) error {
	return o.runner(ctx, o.resolvedConfig)
}
