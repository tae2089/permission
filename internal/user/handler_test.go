package user

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/tae2089/go-template/internal/http/middleware"
)

func TestHandlerCreatesUserWithoutResponseBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	var stored User
	service := NewService(repositoryFunc(func(_ context.Context, user User) error {
		stored = user
		return nil
	}), fixedID(userID))
	handler := NewHandler(service)
	router := newHandlerTestRouter(handler)

	request := httptest.NewRequest(
		http.MethodPost,
		"/users",
		strings.NewReader(`{"name":" Alice ","email":"ALICE@example.com"}`),
	)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status code = %d, want %d", response.Code, http.StatusCreated)
	}
	if stored.Name != "Alice" {
		t.Errorf("stored name = %q, want %q", stored.Name, "Alice")
	}
	if stored.Email != "alice@example.com" {
		t.Errorf("stored email = %q, want %q", stored.Email, "alice@example.com")
	}
	if location := response.Header().Get("Location"); location != "/users/"+userID.String() {
		t.Errorf("Location = %q, want %q", location, "/users/"+userID.String())
	}
	if response.Body.Len() != 0 {
		t.Errorf("response body = %q, want empty", response.Body.String())
	}
}

func TestHandlerRejectsInvalidJSONInput(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name        string
		contentType string
		body        string
	}{
		{name: "missing content type", body: `{}`},
		{name: "empty body", contentType: "application/json"},
		{name: "unknown field", contentType: "application/json", body: `{"name":"Alice","email":"alice@example.com","role":"admin"}`},
		{name: "multiple documents", contentType: "application/json", body: `{} {}`},
		{name: "body too large", contentType: "application/json", body: `{"name":"` + strings.Repeat("a", 64<<10) + `","email":"alice@example.com"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repositoryCalled := false
			service := NewService(repositoryFunc(func(context.Context, User) error {
				repositoryCalled = true
				return nil
			}), fixedID(uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")))
			handler := NewHandler(service)
			router := newHandlerTestRouter(handler)
			request := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(tt.body))
			if tt.contentType != "" {
				request.Header.Set("Content-Type", tt.contentType)
			}
			response := httptest.NewRecorder()

			router.ServeHTTP(response, request)

			if response.Code != http.StatusBadRequest {
				t.Errorf("status code = %d, want %d", response.Code, http.StatusBadRequest)
			}
			if repositoryCalled {
				t.Error("repository was called for invalid input")
			}
		})
	}
}

func newHandlerTestRouter(handler *Handler) *gin.Engine {
	router := gin.New()
	router.Use(middleware.ErrorHandler())
	RegisterRoutes(router.Group("/users"), handler)
	return router
}
