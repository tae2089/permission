package telemetry

import (
	"context"
	"errors"
	"strings"

	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.39.0"
	oteltrace "go.opentelemetry.io/otel/trace"
)

type Options struct {
	ServiceName string
}

type Provider struct {
	serviceName    string
	tracerProvider *sdktrace.TracerProvider
	propagator     propagation.TextMapPropagator
}

func New(opts Options) (*Provider, error) {
	serviceName := strings.TrimSpace(opts.ServiceName)
	if serviceName == "" {
		return nil, errors.New("telemetry service name is required")
	}

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.NeverSample()),
		sdktrace.WithResource(resource.NewSchemaless(semconv.ServiceName(serviceName))),
	)
	propagator := propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)

	return &Provider{
		serviceName:    serviceName,
		tracerProvider: tracerProvider,
		propagator:     propagator,
	}, nil
}

func (p *Provider) ServiceName() string {
	return p.serviceName
}

func (p *Provider) TracerProvider() oteltrace.TracerProvider {
	return p.tracerProvider
}

func (p *Provider) Propagator() propagation.TextMapPropagator {
	return p.propagator
}

func (p *Provider) Shutdown(ctx context.Context) error {
	return p.tracerProvider.Shutdown(ctx)
}
