package middleware

import (
	stderrors "errors"
	"net/http"
	"syscall"

	"github.com/gin-gonic/gin"

	"github.com/tae2089/go-template/internal/http/errors"
	"github.com/tae2089/go-template/internal/telemetry"
)

func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		lastError := c.Errors.Last()
		if lastError == nil {
			return
		}

		err := lastError.Err
		recordError(c.Request.Context(), err)
		if isBrokenConnection(err) || c.Writer.Written() {
			return
		}

		c.AbortWithStatusJSON(errors.Resolve(err, telemetry.TraceID(c.Request.Context())))
	}
}

func isBrokenConnection(err error) bool {
	return stderrors.Is(err, syscall.EPIPE) ||
		stderrors.Is(err, syscall.ECONNRESET) ||
		stderrors.Is(err, http.ErrAbortHandler)
}
