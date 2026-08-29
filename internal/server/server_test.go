package server

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/tae2089/go-template/internal/config"
)

func TestRunReturnsListenerErrorAfterStartup(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	err := Run(context.Background(), config.Config{
		Serve: config.Serve{
			Address: "invalid",
		},
		Database: config.Database{
			Driver: "sqlite",
			DSN:    "file:server-run-test?mode=memory&cache=shared",
		},
	}, logger)
	if err == nil {
		t.Fatal("Run() error = nil, want listener error")
	}
}

func TestNewHTTPServerConfiguresBoundedTimeouts(t *testing.T) {
	server := newHTTPServer(http.NotFoundHandler())
	tests := []struct {
		name string
		got  time.Duration
		want time.Duration
	}{
		{name: "read header", got: server.ReadHeaderTimeout, want: 5 * time.Second},
		{name: "read", got: server.ReadTimeout, want: 60 * time.Second},
		{name: "write", got: server.WriteTimeout, want: 60 * time.Second},
		{name: "idle", got: server.IdleTimeout, want: 120 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("timeout = %s, want %s", tt.got, tt.want)
			}
		})
	}
}

func TestServeStopsWhenContextCanceled(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	serveResult := make(chan error, 1)
	go func() {
		serveResult <- Serve(ctx, listener, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}))
	}()

	client := &http.Client{Timeout: time.Second}
	response, err := client.Get("http://" + listener.Addr().String())
	if err != nil {
		t.Fatalf("get served request: %v", err)
	}
	if err := response.Body.Close(); err != nil {
		t.Fatalf("close response body: %v", err)
	}
	if response.StatusCode != http.StatusNoContent {
		t.Fatalf("status code = %d, want %d", response.StatusCode, http.StatusNoContent)
	}

	cancel()

	select {
	case err := <-serveResult:
		if err != nil {
			t.Fatalf("Serve() error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Serve() did not return after context cancellation")
	}
}

func TestListenAndServeReturnsListenerError(t *testing.T) {
	err := ListenAndServe(context.Background(), "invalid", http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	if err == nil {
		t.Fatal("ListenAndServe() error = nil, want listener error")
	}
}

func TestServeReturnsErrorWhenListenerStopsUnexpectedly(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	serveResult := make(chan error, 1)
	go func() {
		serveResult <- Serve(context.Background(), listener, http.NotFoundHandler())
	}()

	if err := listener.Close(); err != nil {
		t.Fatalf("close listener: %v", err)
	}

	select {
	case err := <-serveResult:
		if err == nil {
			t.Fatal("Serve() error = nil, want listener failure")
		}
	case <-time.After(time.Second):
		t.Fatal("Serve() did not return after listener failure")
	}
}
