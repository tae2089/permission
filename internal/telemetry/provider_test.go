package telemetry_test

import (
	"context"
	"testing"

	"github.com/tae2089/go-template/internal/telemetry"
)

func TestProviderCreatesValidNonRecordingSpanByDefault(t *testing.T) {
	provider, err := telemetry.New(telemetry.Options{ServiceName: "test-service"})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(func() {
		if err := provider.Shutdown(context.Background()); err != nil {
			t.Errorf("Shutdown() error = %v", err)
		}
	})

	_, span := provider.TracerProvider().Tracer("test").Start(context.Background(), "operation")
	defer span.End()

	if !span.SpanContext().IsValid() {
		t.Fatal("span context is invalid")
	}
	if span.IsRecording() {
		t.Error("span is recording when export is disabled")
	}
}

func TestProviderRejectsEmptyServiceName(t *testing.T) {
	_, err := telemetry.New(telemetry.Options{})
	if err == nil {
		t.Fatal("New() error = nil, want service name error")
	}
}
