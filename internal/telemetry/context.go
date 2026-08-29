package telemetry

import (
	"context"

	oteltrace "go.opentelemetry.io/otel/trace"
)

func TraceID(ctx context.Context) string {
	spanContext := oteltrace.SpanContextFromContext(ctx)
	if !spanContext.HasTraceID() {
		return ""
	}
	return spanContext.TraceID().String()
}

func SpanID(ctx context.Context) string {
	spanContext := oteltrace.SpanContextFromContext(ctx)
	if !spanContext.HasSpanID() {
		return ""
	}
	return spanContext.SpanID().String()
}
