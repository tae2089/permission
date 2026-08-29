package middleware

import (
	stderrors "errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/tae2089/go-template/internal/telemetry"
)

func RequestLogger(logger *slog.Logger) gin.HandlerFunc {
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}

	return func(c *gin.Context) {
		startedAt := time.Now()
		c.Next()

		status := c.Writer.Status()
		attrs := []slog.Attr{
			slog.String("trace_id", telemetry.TraceID(c.Request.Context())),
			slog.String("span_id", telemetry.SpanID(c.Request.Context())),
			slog.String("method", c.Request.Method),
			slog.String("route", c.FullPath()),
			slog.Int("status", status),
			slog.Duration("duration", time.Since(startedAt)),
			slog.Int("response_bytes", c.Writer.Size()),
		}
		if lastError := c.Errors.Last(); lastError != nil {
			attrs = append(attrs, errorAttr(lastError.Err))
		}

		logger.LogAttrs(
			c.Request.Context(),
			requestLogLevel(c, status),
			"request completed",
			attrs...,
		)
	}
}

func errorAttr(err error) slog.Attr {
	traceErr := err
	var panicErr *panicError
	if stderrors.As(err, &panicErr) {
		traceErr = panicErr.err
	}

	attrs := []slog.Attr{
		slog.String("message", err.Error()),
		slog.String("trace", fmt.Sprintf("%+v", traceErr)),
	}
	if panicErr != nil {
		attrs = append(attrs, slog.String("panic_stack", panicErr.stack))
	}
	return slog.Attr{Key: "error", Value: slog.GroupValue(attrs...)}
}

func requestLogLevel(c *gin.Context, status int) slog.Level {
	switch {
	case status >= http.StatusInternalServerError:
		return slog.LevelError
	case c.FullPath() == "/healthz" && status < http.StatusBadRequest:
		return slog.LevelDebug
	default:
		return slog.LevelInfo
	}
}
