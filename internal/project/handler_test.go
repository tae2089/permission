package project

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/tae2089/go-template/internal/http/middleware"
)

const testInstanceAdminKey = "test-instance-admin-key"

func TestHandlerCreatesProject(t *testing.T) {
	gin.SetMode(gin.TestMode)
	projectID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	createdAt := time.Date(2026, time.August, 29, 10, 0, 0, 0, time.UTC)
	var gotInput CreateInput
	handler := NewHandler(handlerService{
		create: func(_ context.Context, input CreateInput) (Project, error) {
			gotInput = input
			return Project{ID: projectID, Name: input.Name, CreatedAt: createdAt}, nil
		},
	}, testInstanceAdminKey)
	router := newHandlerTestRouter(handler)
	request := httptest.NewRequest(
		http.MethodPost,
		"/v1/projects",
		strings.NewReader(`{"name":"billing-api"}`),
	)
	request.Header.Set("Authorization", "Bearer "+testInstanceAdminKey)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status code = %d, want %d", response.Code, http.StatusCreated)
	}
	if gotInput.Name != "billing-api" {
		t.Errorf("Create() input = %#v, want name billing-api", gotInput)
	}
	if location := response.Header().Get("Location"); location != "/v1/projects/"+projectID.String() {
		t.Errorf("Location = %q, want /v1/projects/%s", location, projectID)
	}
	var body projectResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.ID != projectID.String() || body.Name != "billing-api" || !body.CreatedAt.Equal(createdAt) {
		t.Errorf("response body = %#v, want created project", body)
	}
}

func TestHandlerListsProjects(t *testing.T) {
	gin.SetMode(gin.TestMode)
	project := Project{
		ID:        uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		Name:      "billing-api",
		CreatedAt: time.Date(2026, time.August, 29, 10, 0, 0, 0, time.UTC),
	}
	handler := NewHandler(handlerService{
		list: func(context.Context) ([]Project, error) {
			return []Project{project}, nil
		},
	}, testInstanceAdminKey)
	router := newHandlerTestRouter(handler)
	request := httptest.NewRequest(http.MethodGet, "/v1/projects", nil)
	request.Header.Set("Authorization", "Bearer "+testInstanceAdminKey)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", response.Code, http.StatusOK)
	}
	var body projectsResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Projects) != 1 || body.Projects[0].ID != project.ID.String() || body.Projects[0].Name != project.Name || !body.Projects[0].CreatedAt.Equal(project.CreatedAt) {
		t.Errorf("response body = %#v, want %#v", body, project)
	}
}

func TestHandlerRejectsUnauthenticatedRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name          string
		authorization string
	}{
		{name: "missing"},
		{name: "wrong scheme", authorization: "Basic ignored"},
		{name: "invalid token", authorization: "Bearer invalid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serviceCalled := false
			handler := NewHandler(handlerService{
				create: func(context.Context, CreateInput) (Project, error) {
					serviceCalled = true
					return Project{}, nil
				},
			}, testInstanceAdminKey)
			router := newHandlerTestRouter(handler)
			request := httptest.NewRequest(http.MethodPost, "/v1/projects", strings.NewReader(`{"name":"billing-api"}`))
			request.Header.Set("Content-Type", "application/json")
			if tt.authorization != "" {
				request.Header.Set("Authorization", tt.authorization)
			}
			response := httptest.NewRecorder()

			router.ServeHTTP(response, request)

			if response.Code != http.StatusUnauthorized {
				t.Errorf("status code = %d, want %d", response.Code, http.StatusUnauthorized)
			}
			if serviceCalled {
				t.Error("service was called without valid instance administrator credentials")
			}
		})
	}
}

func TestHandlerRejectsInvalidJSONInput(t *testing.T) {
	gin.SetMode(gin.TestMode)

	serviceCalled := false
	handler := NewHandler(handlerService{
		create: func(context.Context, CreateInput) (Project, error) {
			serviceCalled = true
			return Project{}, nil
		},
	}, testInstanceAdminKey)
	router := newHandlerTestRouter(handler)
	request := httptest.NewRequest(http.MethodPost, "/v1/projects", strings.NewReader(`{"name":"billing-api","unknown":true}`))
	request.Header.Set("Authorization", "Bearer "+testInstanceAdminKey)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", response.Code, http.StatusBadRequest)
	}
	if serviceCalled {
		t.Error("service was called for invalid input")
	}
}

func newHandlerTestRouter(handler *Handler) *gin.Engine {
	router := gin.New()
	router.Use(middleware.ErrorHandler())
	RegisterRoutes(router.Group("/v1/projects"), handler)
	return router
}

type handlerService struct {
	create func(context.Context, CreateInput) (Project, error)
	list   func(context.Context) ([]Project, error)
}

func (s handlerService) Create(ctx context.Context, input CreateInput) (Project, error) {
	if s.create == nil {
		return Project{}, nil
	}
	return s.create(ctx, input)
}

func (s handlerService) List(ctx context.Context) ([]Project, error) {
	if s.list == nil {
		return nil, nil
	}
	return s.list(ctx)
}
