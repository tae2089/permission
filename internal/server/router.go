package server

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

	"github.com/tae2089/go-template/internal/health"
	"github.com/tae2089/go-template/internal/http/middleware"
	"github.com/tae2089/go-template/internal/project"
	"github.com/tae2089/go-template/internal/telemetry"
	"github.com/tae2089/go-template/internal/user"
)

func New(
	logger *slog.Logger,
	telemetryProvider *telemetry.Provider,
	healthHandler *health.Handler,
	userHandler *user.Handler,
	projectHandler *project.Handler,
) *gin.Engine {
	router := gin.New()
	router.Use(
		otelgin.Middleware(
			telemetryProvider.ServiceName(),
			otelgin.WithTracerProvider(telemetryProvider.TracerProvider()),
			otelgin.WithPropagators(telemetryProvider.Propagator()),
		),
		middleware.RequestLogger(logger),
		middleware.ErrorHandler(),
		middleware.Recovery(),
	)

	health.RegisterRoutes(router.Group("/healthz"), healthHandler)
	user.RegisterRoutes(router.Group("/users"), userHandler)
	project.RegisterRoutes(router.Group("/v1/projects"), projectHandler)

	return router
}
