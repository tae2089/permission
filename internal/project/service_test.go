package project

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/tae2089/go-template/internal/apperr"
	"github.com/tae2089/go-template/internal/audit"
)

func TestServiceCreatesProject(t *testing.T) {
	projectID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	auditEventID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440001")
	createdAt := time.Date(2026, time.August, 29, 10, 0, 0, 0, time.UTC)
	var stored Project
	var storedAuditEvent audit.Event
	ids := []uuid.UUID{projectID, auditEventID}
	service := NewService(
		repositoryFunc(func(_ context.Context, project Project, auditEvent audit.Event) error {
			stored = project
			storedAuditEvent = auditEvent
			return nil
		}),
		func() (uuid.UUID, error) {
			id := ids[0]
			ids = ids[1:]
			return id, nil
		},
		func() time.Time {
			return createdAt
		},
	)

	created, err := service.Create(context.Background(), CreateInput{Name: "billing-api"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.ID != projectID {
		t.Errorf("created ID = %s, want %s", created.ID, projectID)
	}
	if created.Name != "billing-api" {
		t.Errorf("created Name = %q, want %q", created.Name, "billing-api")
	}
	if !created.CreatedAt.Equal(createdAt) {
		t.Errorf("created CreatedAt = %s, want %s", created.CreatedAt, createdAt)
	}
	if stored != created {
		t.Errorf("stored project = %#v, want %#v", stored, created)
	}
	wantAuditEvent := audit.Event{
		ID:         auditEventID,
		OccurredAt: createdAt,
		Action:     auditActionProjectCreated,
		Actor:      auditActorInstanceAdmin,
		ProjectID:  projectID,
		TargetID:   projectID.String(),
	}
	if storedAuditEvent != wantAuditEvent {
		t.Errorf("stored audit event = %#v, want %#v", storedAuditEvent, wantAuditEvent)
	}
}

func TestServiceRejectsInvalidProjectName(t *testing.T) {
	tests := []struct {
		name        string
		projectName string
	}{
		{name: "empty", projectName: ""},
		{name: "uppercase", projectName: "Billing"},
		{name: "space", projectName: "billing api"},
		{name: "Korean", projectName: "청구"},
		{name: "too long", projectName: strings.Repeat("a", 64)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repositoryCalled := false
			idGeneratorCalled := false
			service := NewService(
				repositoryFunc(func(context.Context, Project, audit.Event) error {
					repositoryCalled = true
					return nil
				}),
				func() (uuid.UUID, error) {
					idGeneratorCalled = true
					return uuid.New(), nil
				},
				time.Now,
			)

			_, err := service.Create(context.Background(), CreateInput{Name: tt.projectName})

			var target *apperr.Error
			if !errors.As(err, &target) || target.Kind != apperr.KindBadParameter {
				t.Errorf("Create() error = %v, want bad parameter", err)
			}
			if repositoryCalled {
				t.Error("repository was called for an invalid project name")
			}
			if idGeneratorCalled {
				t.Error("ID generator was called for an invalid project name")
			}
		})
	}
}

func TestServiceListsProjects(t *testing.T) {
	want := []Project{
		{
			ID:        uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
			Name:      "billing-api",
			CreatedAt: time.Date(2026, time.August, 29, 10, 0, 0, 0, time.UTC),
		},
	}
	service := NewService(
		listRepository{projects: want},
		uuid.NewRandom,
		time.Now,
	)

	got, err := service.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(got) != len(want) || got[0] != want[0] {
		t.Errorf("List() = %#v, want %#v", got, want)
	}
}

type repositoryFunc func(context.Context, Project, audit.Event) error

func (f repositoryFunc) Create(ctx context.Context, project Project, auditEvent audit.Event) error {
	return f(ctx, project, auditEvent)
}

func (repositoryFunc) List(context.Context) ([]Project, error) {
	return nil, nil
}

type listRepository struct {
	projects []Project
}

func (r listRepository) Create(context.Context, Project, audit.Event) error {
	return nil
}

func (r listRepository) List(context.Context) ([]Project, error) {
	return r.projects, nil
}
