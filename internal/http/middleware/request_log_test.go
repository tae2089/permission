package middleware_test

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/tae2089/trace/v3"
	oteltrace "go.opentelemetry.io/otel/trace"

	"github.com/tae2089/go-template/internal/apperr"
	"github.com/tae2089/go-template/internal/http/middleware"
)

func TestRequestLoggerWritesOneStructuredCompletionLog(t *testing.T) {
	var output strings.Builder
	logger := slog.New(slog.NewJSONHandler(&output, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	provider := newTelemetryProvider(t)
	router := gin.New()
	router.Use(
		tracingMiddleware(provider),
		middleware.RequestLogger(logger),
		middleware.ErrorHandler(),
	)
	router.GET("/users/:id", func(c *gin.Context) {
		err := trace.Wrap(apperr.New(apperr.KindNotFound, "user not found"), "load user")
		c.Error(err) //nolint:errcheck
		c.Abort()
	})

	request := httptest.NewRequest(http.MethodGet, "/users/42", nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 1 {
		t.Fatalf("log line count = %d, want 1\n%s", len(lines), output.String())
	}

	var record map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &record); err != nil {
		t.Fatalf("decode log: %v", err)
	}

	assertLogValue(t, record, "level", "INFO")
	assertLogValue(t, record, "msg", "request completed")
	traceID, ok := record["trace_id"].(string)
	if !ok {
		t.Fatalf("trace_id = %#v, want string", record["trace_id"])
	}
	if _, err := oteltrace.TraceIDFromHex(traceID); err != nil {
		t.Errorf("trace_id = %q, want valid trace ID: %v", traceID, err)
	}
	spanID, ok := record["span_id"].(string)
	if !ok {
		t.Fatalf("span_id = %#v, want string", record["span_id"])
	}
	if _, err := oteltrace.SpanIDFromHex(spanID); err != nil {
		t.Errorf("span_id = %q, want valid span ID: %v", spanID, err)
	}
	assertLogValue(t, record, "method", http.MethodGet)
	assertLogValue(t, record, "route", "/users/:id")
	assertLogNumber(t, record, "status", http.StatusNotFound)

	if _, ok := record["duration"]; !ok {
		t.Error("duration attribute is missing")
	}
	if size, ok := record["response_bytes"].(float64); !ok || size <= 0 {
		t.Errorf("response_bytes = %#v, want positive number", record["response_bytes"])
	}
	errorRecord, ok := record["error"].(map[string]any)
	if !ok {
		t.Errorf("error = %#v, want structured object", record["error"])
		return
	}
	assertLogValue(t, errorRecord, "message", "load user: user not found")
	traceOutput, ok := errorRecord["trace"].(string)
	if !ok {
		t.Fatalf("error.trace = %#v, want string", errorRecord["trace"])
	}
	for _, want := range []string{"trace: load user", "trace: origin"} {
		if !strings.Contains(traceOutput, want) {
			t.Errorf("error.trace = %q, want %q", traceOutput, want)
		}
	}
}

func assertLogValue(t *testing.T, record map[string]any, key string, want any) {
	t.Helper()
	if got := record[key]; got != want {
		t.Errorf("%s = %#v, want %#v", key, got, want)
	}
}

func assertLogNumber(t *testing.T, record map[string]any, key string, want int) {
	t.Helper()
	got, ok := record[key].(float64)
	if !ok || int(got) != want {
		t.Errorf("%s = %#v, want %d", key, record[key], want)
	}
}
