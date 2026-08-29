package middleware

import (
	"context"

	oteltrace "go.opentelemetry.io/otel/trace"
)

func recordError(ctx context.Context, err error) {
	oteltrace.SpanFromContext(ctx).RecordError(err)
}
