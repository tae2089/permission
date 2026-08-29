package apikey

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

func TestHandlerIssuesProjectAdministratorKeyForInstanceAdministrator(t *testing.T) {
	gin.SetMode(gin.TestMode)
	projectID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	key := Key{ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"), ProjectID: projectID, Kind: KindProjectAdmin, Status: StatusActive, CreatedAt: time.Date(2026, time.August, 30, 1, 0, 0, 0, time.UTC)}
	var actor any
	handler := NewHandler(handlerService{issue: func(_ context.Context, gotActor any, input IssueInput) (IssuedKey, error) {
		actor = gotActor
		if input.ProjectID != projectID || input.Kind != KindProjectAdmin {
			t.Errorf("Issue input = %#v", input)
		}
		return IssuedKey{Key: key, Secret: "returned-once"}, nil
	}}, testInstanceAdminKey)
	router := gin.New()
	router.Use(middleware.ErrorHandler())
	RegisterRoutes(router.Group("/v1"), handler)
	request := httptest.NewRequest(http.MethodPost, "/v1/projects/"+projectID.String()+"/api-keys", strings.NewReader(`{"kind":"project_admin"}`))
	request.Header.Set("Authorization", "Bearer "+testInstanceAdminKey)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status code = %d, want %d", response.Code, http.StatusCreated)
	}
	if _, ok := actor.(InstanceAdministrator); !ok {
		t.Errorf("actor = %T, want InstanceAdministrator", actor)
	}
	var body issuedKeyResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Secret != "returned-once" || body.ID != key.ID.String() || body.SecretHash != "" {
		t.Errorf("response = %#v, want public key metadata and one-time secret", body)
	}
}

type handlerService struct {
	issue func(context.Context, any, IssueInput) (IssuedKey, error)
}

func (s handlerService) Issue(ctx context.Context, actor any, input IssueInput) (IssuedKey, error) {
	return s.issue(ctx, actor, input)
}
func (handlerService) Authenticate(context.Context, string) (Key, error)   { return Key{}, nil }
func (handlerService) List(context.Context, any, uuid.UUID) ([]Key, error) { return nil, nil }
func (handlerService) Rotate(context.Context, ProjectAdministrator, uuid.UUID, uuid.UUID) (IssuedKey, error) {
	return IssuedKey{}, nil
}
func (handlerService) Revoke(context.Context, ProjectAdministrator, uuid.UUID, uuid.UUID) error {
	return nil
}
