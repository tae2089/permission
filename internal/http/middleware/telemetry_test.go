package middleware_test

import (
	"context"
	"testing"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

	"github.com/tae2089/go-template/internal/telemetry"
)

func newTelemetryProvider(t *testing.T) *telemetry.Provider {
	t.Helper()

	provider, err := telemetry.New(telemetry.Options{ServiceName: "test-service"})
	if err != nil {
		t.Fatalf("create telemetry provider: %v", err)
	}
	t.Cleanup(func() {
		if err := provider.Shutdown(context.Background()); err != nil {
			t.Errorf("shutdown telemetry provider: %v", err)
		}
	})
	return provider
}

func tracingMiddleware(provider *telemetry.Provider) gin.HandlerFunc {
	return otelgin.Middleware(
		provider.ServiceName(),
		otelgin.WithTracerProvider(provider.TracerProvider()),
		otelgin.WithPropagators(provider.Propagator()),
	)
}
