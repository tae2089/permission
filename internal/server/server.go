package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/tae2089/go-template/internal/config"
	"github.com/tae2089/go-template/internal/database"
	"github.com/tae2089/go-template/internal/health"
	"github.com/tae2089/go-template/internal/telemetry"
	"github.com/tae2089/go-template/internal/user"
)

const (
	ApplicationName   = "app"
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 60 * time.Second
	writeTimeout      = 60 * time.Second
	idleTimeout       = 120 * time.Second
	shutdownTimeout   = 10 * time.Second
)

func Run(ctx context.Context, cfg config.Config, logger *slog.Logger) error {
	connection, err := database.Open(ctx, database.Options{
		Driver: database.Driver(cfg.Database.Driver),
		DSN:    cfg.Database.DSN,
	})
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}

	provider, err := telemetry.New(telemetry.Options{ServiceName: ApplicationName})
	if err != nil {
		return errors.Join(
			fmt.Errorf("create telemetry provider: %w", err),
			connection.Close(),
		)
	}

	userRepository := user.NewRepository(connection.DB())
	userService := user.NewService(userRepository, uuid.NewRandom)
	router := New(logger, provider, health.NewHandler(), user.NewHandler(userService))
	logger.InfoContext(ctx, "server starting", "address", cfg.Serve.Address)

	serveErr := ListenAndServe(ctx, cfg.Serve.Address, router)
	if serveErr == nil {
		logger.InfoContext(context.Background(), "server stopped")
	}

	cleanupCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	shutdownErr := provider.Shutdown(cleanupCtx)
	if shutdownErr != nil {
		shutdownErr = fmt.Errorf("shutdown telemetry provider: %w", shutdownErr)
	}

	return errors.Join(serveErr, shutdownErr, connection.Close())
}

func ListenAndServe(ctx context.Context, address string, handler http.Handler) error {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", address, err)
	}

	return Serve(ctx, listener, handler)
}

func Serve(ctx context.Context, listener net.Listener, handler http.Handler) error {
	httpServer := newHTTPServer(handler)
	serveResult := make(chan error, 1)
	go func() {
		serveResult <- httpServer.Serve(listener)
	}()

	select {
	case err := <-serveResult:
		return serveError(err)
	case <-ctx.Done():
		return shutdown(httpServer, serveResult)
	}
}

func newHTTPServer(handler http.Handler) *http.Server {
	return &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}
}

func shutdown(httpServer *http.Server, serveResult <-chan error) error {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		closeErr := httpServer.Close()
		serveErr := <-serveResult
		return errors.Join(
			fmt.Errorf("shutdown http server: %w", err),
			closeError(closeErr),
			serveError(serveErr),
		)
	}

	return serveError(<-serveResult)
}

func closeError(err error) error {
	if err == nil || errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return fmt.Errorf("force close http server: %w", err)
}

func serveError(err error) error {
	if err == nil || errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return fmt.Errorf("serve http server: %w", err)
}
