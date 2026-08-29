package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/tae2089/go-template/internal/audit"
	"github.com/tae2089/go-template/internal/database"
	"github.com/tae2089/go-template/internal/health"
	httperrors "github.com/tae2089/go-template/internal/http/errors"
	"github.com/tae2089/go-template/internal/project"
	"github.com/tae2089/go-template/internal/telemetry"
	"github.com/tae2089/go-template/internal/user"
)

func TestRouterServesHealthCheck(t *testing.T) {
	var logs strings.Builder
	logger := slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	provider, err := telemetry.New(telemetry.Options{ServiceName: "test-service"})
	if err != nil {
		t.Fatalf("create telemetry provider: %v", err)
	}
	t.Cleanup(func() {
		if err := provider.Shutdown(context.Background()); err != nil {
			t.Errorf("shutdown telemetry provider: %v", err)
		}
	})

	userService := user.NewService(
		stubRepository{},
		func() (uuid.UUID, error) {
			return uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), nil
		},
	)
	router := New(logger, provider, health.NewHandler(), user.NewHandler(userService), newProjectHandler())
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", response.Code, http.StatusOK)
	}
	if got := response.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q, want %q", got, "application/json; charset=utf-8")
	}
	if got := response.Header().Get("X-Request-ID"); got != "" {
		t.Errorf("X-Request-ID = %q, want empty", got)
	}

	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Status != "ok" {
		t.Errorf("status = %q, want %q", body.Status, "ok")
	}
	if got := strings.Count(strings.TrimSpace(logs.String()), "\n") + 1; got != 1 {
		t.Errorf("log line count = %d, want 1\n%s", got, logs.String())
	}
	if !strings.Contains(logs.String(), `"level":"DEBUG"`) {
		t.Errorf("health log level is not DEBUG: %s", logs.String())
	}
	if !strings.Contains(logs.String(), `"route":"/healthz"`) {
		t.Errorf("health log route is missing: %s", logs.String())
	}
	if !strings.Contains(logs.String(), `"trace_id":"`) {
		t.Errorf("health log trace ID is missing: %s", logs.String())
	}
}

type stubRepository struct{}

func (stubRepository) Create(context.Context, user.User) error {
	return nil
}

func TestRouterCreatesUserThroughSQLite(t *testing.T) {
	ctx := context.Background()
	connection, err := database.Open(ctx, database.Options{
		Driver: database.DriverSQLite,
		DSN:    "file:" + uuid.NewString() + "?mode=memory&cache=shared",
	})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() {
		if err := connection.Close(); err != nil {
			t.Errorf("close database: %v", err)
		}
	})
	if err := user.Migrate(ctx, connection.DB()); err != nil {
		t.Fatalf("migrate users: %v", err)
	}

	provider, err := telemetry.New(telemetry.Options{ServiceName: "test-service"})
	if err != nil {
		t.Fatalf("create telemetry provider: %v", err)
	}
	t.Cleanup(func() {
		if err := provider.Shutdown(context.Background()); err != nil {
			t.Errorf("shutdown telemetry provider: %v", err)
		}
	})
	repository := user.NewRepository(connection.DB())
	service := user.NewService(
		repository,
		func() (uuid.UUID, error) {
			return uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), nil
		},
	)
	router := New(
		slog.New(slog.DiscardHandler),
		provider,
		health.NewHandler(),
		user.NewHandler(service),
		newProjectHandler(),
	)
	request := httptest.NewRequest(
		http.MethodPost,
		"/users",
		strings.NewReader(`{"name":" Alice ","email":"ALICE@example.com"}`),
	)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status code = %d, want %d: %s", response.Code, http.StatusCreated, response.Body.String())
	}
	if response.Body.Len() != 0 {
		t.Errorf("response body = %q, want empty", response.Body.String())
	}
	location := response.Header().Get("Location")
	if _, err := uuid.Parse(strings.TrimPrefix(location, "/users/")); err != nil {
		t.Errorf("Location = %q, want /users/<uuid>: %v", location, err)
	}

	duplicateRequest := httptest.NewRequest(
		http.MethodPost,
		"/users",
		strings.NewReader(`{"name":"Another Alice","email":"alice@EXAMPLE.com"}`),
	)
	duplicateRequest.Header.Set("Content-Type", "application/json")
	duplicateResponse := httptest.NewRecorder()
	router.ServeHTTP(duplicateResponse, duplicateRequest)

	if duplicateResponse.Code != http.StatusConflict {
		t.Fatalf(
			"duplicate status code = %d, want %d: %s",
			duplicateResponse.Code,
			http.StatusConflict,
			duplicateResponse.Body.String(),
		)
	}
	var errorResponse httperrors.Response
	if err := json.NewDecoder(duplicateResponse.Body).Decode(&errorResponse); err != nil {
		t.Fatalf("decode duplicate response: %v", err)
	}
	if errorResponse.Error.Code != httperrors.CodeAlreadyExists {
		t.Errorf(
			"duplicate error code = %q, want %q",
			errorResponse.Error.Code,
			httperrors.CodeAlreadyExists,
		)
	}

	var count int64
	if err := connection.DB().Table("users").
		Where("name = ? AND email = ?", "Alice", "alice@example.com").
		Count(&count).Error; err != nil {
		t.Fatalf("count created users: %v", err)
	}
	if count != 1 {
		t.Errorf("created user count = %d, want 1", count)
	}
}

func TestRouterCreatesProjectAndAuditEventThroughSQLite(t *testing.T) {
	ctx := context.Background()
	connection, err := database.Open(ctx, database.Options{
		Driver: database.DriverSQLite,
		DSN:    "file:" + uuid.NewString() + "?mode=memory&cache=shared",
	})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() {
		if err := connection.Close(); err != nil {
			t.Errorf("close database: %v", err)
		}
	})
	if err := audit.Migrate(ctx, connection.DB()); err != nil {
		t.Fatalf("migrate audit events: %v", err)
	}
	if err := project.Migrate(ctx, connection.DB()); err != nil {
		t.Fatalf("migrate projects: %v", err)
	}

	provider, err := telemetry.New(telemetry.Options{ServiceName: "test-service"})
	if err != nil {
		t.Fatalf("create telemetry provider: %v", err)
	}
	t.Cleanup(func() {
		if err := provider.Shutdown(context.Background()); err != nil {
			t.Errorf("shutdown telemetry provider: %v", err)
		}
	})
	projectService := project.NewService(project.NewRepository(connection.DB(), audit.NewWriter()), uuid.NewRandom, time.Now)
	router := New(
		slog.New(slog.DiscardHandler),
		provider,
		health.NewHandler(),
		user.NewHandler(user.NewService(stubRepository{}, uuid.NewRandom)),
		project.NewHandler(projectService, "test-instance-admin-key"),
	)
	request := httptest.NewRequest(
		http.MethodPost,
		"/v1/projects",
		strings.NewReader(`{"name":"billing-api"}`),
	)
	request.Header.Set("Authorization", "Bearer test-instance-admin-key")
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status code = %d, want %d: %s", response.Code, http.StatusCreated, response.Body.String())
	}
	var projectCount int64
	if err := connection.DB().Table("projects").Count(&projectCount).Error; err != nil {
		t.Fatalf("count projects: %v", err)
	}
	if projectCount != 1 {
		t.Errorf("project count = %d, want 1", projectCount)
	}
	var auditEventCount int64
	if err := connection.DB().Table("audit_events").Where("action = ?", "project.created").Count(&auditEventCount).Error; err != nil {
		t.Fatalf("count project audit events: %v", err)
	}
	if auditEventCount != 1 {
		t.Errorf("project audit event count = %d, want 1", auditEventCount)
	}
}

func newProjectHandler() *project.Handler {
	service := project.NewService(projectStubRepository{}, uuid.NewRandom, time.Now)
	return project.NewHandler(service, "test-instance-admin-key")
}

type projectStubRepository struct{}

func (projectStubRepository) Create(context.Context, project.Project, audit.Event) error {
	return nil
}

func (projectStubRepository) List(context.Context) ([]project.Project, error) {
	return nil, nil
}
