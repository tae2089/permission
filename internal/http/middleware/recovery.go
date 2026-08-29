package middleware

import (
	"io"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"github.com/tae2089/trace/v3"
)

func Recovery() gin.HandlerFunc {
	return gin.CustomRecoveryWithWriter(io.Discard, func(c *gin.Context, recovered any) {
		err := &panicError{
			err:   trace.Errorf("panic recovered: %v", recovered),
			stack: string(debug.Stack()),
		}

		c.Error(err) //nolint:errcheck
		c.Abort()
	})
}

type panicError struct {
	err   error
	stack string
}

func (e *panicError) Error() string {
	return e.err.Error()
}

func (e *panicError) Unwrap() error {
	return e.err
}
