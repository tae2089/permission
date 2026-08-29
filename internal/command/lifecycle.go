package command

import (
	"context"

	"github.com/tae2089/go-template/internal/config"
)

type Lifecycle interface {
	Complete(config.Options) error
	Validate() error
	Run(context.Context) error
}
