package middleware_test

import (
	"encoding/json"
	stderrors "errors"
	"net/http"
	"net/http/httptest"
	"syscall"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/tae2089/trace/v3"
	oteltrace "go.opentelemetry.io/otel/trace"

	"github.com/tae2089/go-template/internal/apperr"
	"github.com/tae2089/go-template/internal/http/errors"
	"github.com/tae2089/go-template/internal/http/middleware"
)

func TestErrorHandlerRendersAttachedTypedErrors(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		wantStatus  int
		wantCode    errors.Code
		wantMessage string
	}{
		{
			name:        "bad parameter",
			err:         trace.Wrap(apperr.New(apperr.KindBadParameter, "invalid name"), "decode request"),
			wantStatus:  http.StatusBadRequest,
			wantCode:    errors.CodeBadRequest,
			wantMessage: "invalid name",
		},
		{
			name:        "not found",
			err:         trace.Wrap(apperr.New(apperr.KindNotFound, "user not found"), "load user"),
			wantStatus:  http.StatusNotFound,
			wantCode:    errors.CodeNotFound,
			wantMessage: "user not found",
		},
		{
			name: "nested layer wraps preserve semantic error",
			err: trace.Wrap(
				trace.Wrap(apperr.New(apperr.KindNotFound, "user not found"), "query user repository"),
				"get user service",
			),
			wantStatus:  http.StatusNotFound,
			wantCode:    errors.CodeNotFound,
			wantMessage: "user not found",
		},
		{
			name:        "already exists",
			err:         trace.Wrap(apperr.New(apperr.KindAlreadyExists, "user already exists"), "create user"),
			wantStatus:  http.StatusConflict,
			wantCode:    errors.CodeAlreadyExists,
			wantMessage: "user already exists",
		},
		{
			name: "nested layer wraps preserve unauthenticated error",
			err: trace.Wrap(
				trace.Wrap(apperr.New(apperr.KindUnauthenticated, "expired bearer token"), "verify token"),
				"authenticate request",
			),
			wantStatus:  http.StatusUnauthorized,
			wantCode:    errors.CodeUnauthenticated,
			wantMessage: "authentication required",
		},
		{
			name:        "access denied",
			err:         trace.Wrap(apperr.New(apperr.KindAccessDenied, "role admin is required"), "authorize user"),
			wantStatus:  http.StatusForbidden,
			wantCode:    errors.CodeAccessDenied,
			wantMessage: "access denied",
		},
		{
			name:        "connection problem",
			err:         apperr.Wrap(apperr.KindUnavailable, stderrors.New("database password leaked"), "query database"),
			wantStatus:  http.StatusServiceUnavailable,
			wantCode:    errors.CodeUnavailable,
			wantMessage: "internal server error",
		},
		{
			name:        "unknown error",
			err:         stderrors.New("database password leaked"),
			wantStatus:  http.StatusInternalServerError,
			wantCode:    errors.CodeInternal,
			wantMessage: "internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := newTelemetryProvider(t)
			router := gin.New()
			router.Use(tracingMiddleware(provider), middleware.ErrorHandler())
			router.GET("/test", func(c *gin.Context) {
				c.Error(tt.err) //nolint:errcheck
				c.Abort()
			})

			request := httptest.NewRequest(http.MethodGet, "/test", nil)
			response := httptest.NewRecorder()

			router.ServeHTTP(response, request)

			if response.Code != tt.wantStatus {
				t.Fatalf("status code = %d, want %d", response.Code, tt.wantStatus)
			}

			var body errors.Response
			if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if body.Error.Code != tt.wantCode {
				t.Errorf("error code = %q, want %q", body.Error.Code, tt.wantCode)
			}
			if body.Error.Message != tt.wantMessage {
				t.Errorf("error message = %q, want %q", body.Error.Message, tt.wantMessage)
			}
			if body.Error.TraceID == "" {
				t.Fatal("trace ID is empty")
			}
			if _, err := oteltrace.TraceIDFromHex(body.Error.TraceID); err != nil {
				t.Errorf("trace ID %q is invalid: %v", body.Error.TraceID, err)
			}
			if got := response.Header().Get("X-Request-ID"); got != "" {
				t.Errorf("X-Request-ID = %q, want empty", got)
			}
		})
	}
}

func TestErrorHandlerContinuesIncomingTraceContext(t *testing.T) {
	const traceID = "0af7651916cd43dd8448eb211c80319c"

	provider := newTelemetryProvider(t)
	router := gin.New()
	router.Use(tracingMiddleware(provider), middleware.ErrorHandler())
	router.GET("/test", func(c *gin.Context) {
		c.Error(apperr.New(apperr.KindNotFound, "user not found")) //nolint:errcheck
		c.Abort()
	})

	request := httptest.NewRequest(http.MethodGet, "/test", nil)
	request.Header.Set("traceparent", "00-"+traceID+"-b7ad6b7169203331-01")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	var body errors.Response
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Error.TraceID != traceID {
		t.Errorf("trace ID = %q, want %q", body.Error.TraceID, traceID)
	}
}

func TestErrorHandlerDoesNotOverwriteStartedResponse(t *testing.T) {
	provider := newTelemetryProvider(t)
	router := gin.New()
	router.Use(tracingMiddleware(provider), middleware.ErrorHandler())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusAccepted, gin.H{"status": "accepted"})
		c.Error(apperr.New(apperr.KindNotFound, "user not found")) //nolint:errcheck
		c.Abort()
	})

	request := httptest.NewRequest(http.MethodGet, "/test", nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusAccepted {
		t.Errorf("status code = %d, want %d", response.Code, http.StatusAccepted)
	}
	if got := response.Body.String(); got != `{"status":"accepted"}` {
		t.Errorf("body = %q, want accepted response", got)
	}
}

func TestErrorHandlerDoesNotWriteBrokenConnectionResponse(t *testing.T) {
	provider := newTelemetryProvider(t)
	router := gin.New()
	router.Use(tracingMiddleware(provider), middleware.ErrorHandler())
	router.GET("/test", func(c *gin.Context) {
		c.Error(trace.Wrap(syscall.EPIPE, "write response")) //nolint:errcheck
		c.Abort()
	})

	request := httptest.NewRequest(http.MethodGet, "/test", nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Body.Len() != 0 {
		t.Errorf("body = %q, want empty", response.Body.String())
	}
	if got := response.Header().Get("Content-Type"); got != "" {
		t.Errorf("Content-Type = %q, want empty", got)
	}
}
