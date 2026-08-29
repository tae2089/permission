package middleware_test

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	oteltrace "go.opentelemetry.io/otel/trace"

	"github.com/tae2089/go-template/internal/http/errors"
	"github.com/tae2089/go-template/internal/http/middleware"
)

func TestRecoveryRendersSafeResponseAndLogsOnce(t *testing.T) {
	const internalMessage = "database password leaked"

	var output strings.Builder
	logger := slog.New(slog.NewJSONHandler(&output, nil))

	provider := newTelemetryProvider(t)
	router := gin.New()
	router.Use(
		tracingMiddleware(provider),
		middleware.RequestLogger(logger),
		middleware.ErrorHandler(),
		middleware.Recovery(),
	)
	router.GET("/panic", func(*gin.Context) {
		panic(internalMessage)
	})

	request := httptest.NewRequest(http.MethodGet, "/panic", nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status code = %d, want %d", response.Code, http.StatusInternalServerError)
	}
	if strings.Contains(response.Body.String(), internalMessage) {
		t.Fatalf("response exposes panic: %s", response.Body.String())
	}

	var body errors.Response
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Error.Code != errors.CodeInternal {
		t.Errorf("error code = %q, want %q", body.Error.Code, errors.CodeInternal)
	}
	if body.Error.Message != "internal server error" {
		t.Errorf("error message = %q, want %q", body.Error.Message, "internal server error")
	}
	if _, err := oteltrace.TraceIDFromHex(body.Error.TraceID); err != nil {
		t.Errorf("trace ID = %q, want valid trace ID: %v", body.Error.TraceID, err)
	}

	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 1 {
		t.Fatalf("log line count = %d, want 1\n%s", len(lines), output.String())
	}

	var record map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &record); err != nil {
		t.Fatalf("decode log: %v", err)
	}
	assertLogValue(t, record, "level", "ERROR")
	assertLogValue(t, record, "trace_id", body.Error.TraceID)
	assertLogNumber(t, record, "status", http.StatusInternalServerError)
	errorRecord, ok := record["error"].(map[string]any)
	if !ok {
		t.Errorf("error = %#v, want structured object", record["error"])
		return
	}
	if _, ok := errorRecord["panic_stack"].(string); !ok {
		t.Errorf("error.panic_stack = %#v, want string", errorRecord["panic_stack"])
	}
	if !strings.Contains(output.String(), internalMessage) {
		t.Error("internal panic is missing from the server log")
	}
}
