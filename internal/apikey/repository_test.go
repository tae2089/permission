package apikey

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/tae2089/go-template/internal/audit"
	"github.com/tae2089/go-template/internal/database"
	"github.com/tae2089/go-template/internal/project"
	"gorm.io/gorm"
)

func TestRepositoryCreatesAndRevokesKeyWithAuditEvents(t *testing.T) {
	ctx := context.Background()
	db := openRepositoryTestDatabase(t, ctx)
	if err := audit.Migrate(ctx, db); err != nil {
		t.Fatalf("migrate audit events: %v", err)
	}
	if err := project.Migrate(ctx, db); err != nil {
		t.Fatalf("migrate projects: %v", err)
	}
	if err := Migrate(ctx, db); err != nil {
		t.Fatalf("migrate API keys: %v", err)
	}
	projectID := uuid.New()
	createdAt := time.Date(2026, time.August, 30, 2, 0, 0, 0, time.UTC)
	projectEvent := audit.Event{ID: uuid.New(), OccurredAt: createdAt, Action: "project.created", Actor: "instance_admin", ProjectID: projectID, TargetID: projectID.String()}
	if err := project.NewRepository(db, audit.NewWriter()).Create(ctx, project.Project{ID: projectID, Name: "billing-api", CreatedAt: createdAt}, projectEvent); err != nil {
		t.Fatalf("create project: %v", err)
	}
	repository := NewRepository(db, audit.NewWriter())
	key := Key{ID: uuid.New(), ProjectID: projectID, Kind: KindProjectAdmin, Status: StatusActive, SecretHash: "derived-hash", CreatedAt: createdAt}
	issuedEvent := audit.Event{ID: uuid.New(), OccurredAt: createdAt, Action: auditActionKeyIssued, Actor: auditActorInstanceAdmin, ProjectID: projectID, TargetID: key.ID.String()}
	if err := repository.Create(ctx, key, issuedEvent); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	second := Key{ID: uuid.New(), ProjectID: projectID, Kind: KindProjectAdmin, Status: StatusActive, SecretHash: "second-derived-hash", CreatedAt: createdAt}
	if err := repository.Create(ctx, second, audit.Event{ID: uuid.New(), OccurredAt: createdAt, Action: auditActionKeyIssued, Actor: auditActorInstanceAdmin, ProjectID: projectID, TargetID: second.ID.String()}); err != nil {
		t.Fatalf("create second key: %v", err)
	}
	third := Key{ID: uuid.New(), ProjectID: projectID, Kind: KindProjectAdmin, Status: StatusActive, SecretHash: "third-derived-hash", CreatedAt: createdAt}
	if err := repository.Create(ctx, third, audit.Event{ID: uuid.New(), OccurredAt: createdAt, Action: auditActionKeyIssued, Actor: auditActorInstanceAdmin, ProjectID: projectID, TargetID: third.ID.String()}); err == nil {
		t.Fatal("create third active key error = nil, want limit exceeded")
	}
	if _, err := repository.FindActiveBySecretHash(ctx, key.SecretHash); err != nil {
		t.Fatalf("FindActiveBySecretHash() error = %v", err)
	}
	revokedAt := createdAt.Add(time.Minute)
	revokedEvent := audit.Event{ID: uuid.New(), OccurredAt: revokedAt, Action: auditActionKeyRevoked, Actor: "project_admin", ProjectID: projectID, TargetID: key.ID.String()}
	if err := repository.Revoke(ctx, projectID, key.ID, revokedEvent); err != nil {
		t.Fatalf("Revoke() error = %v", err)
	}
	if _, err := repository.FindActiveBySecretHash(ctx, key.SecretHash); err == nil {
		t.Fatal("FindActiveBySecretHash() error = nil after revocation")
	}
	var events int64
	if err := db.Table("audit_events").Where("project_id = ? AND action LIKE ?", projectID.String(), "api_key.%").Count(&events).Error; err != nil {
		t.Fatalf("count audit events: %v", err)
	}
	if events != 3 {
		t.Errorf("audit event count = %d, want 3", events)
	}
}

func openRepositoryTestDatabase(t *testing.T, ctx context.Context) *gorm.DB {
	t.Helper()
	connection, err := database.Open(ctx, database.Options{Driver: database.DriverSQLite, DSN: "file:" + uuid.NewString() + "?mode=memory&cache=shared"})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() {
		if err := connection.Close(); err != nil {
			t.Errorf("close database: %v", err)
		}
	})
	return connection.DB()
}
