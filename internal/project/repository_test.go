package project

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/tae2089/go-template/internal/audit"
	"github.com/tae2089/go-template/internal/database"
)

func TestRepositoryCreatesProjectAndAuditEvent(t *testing.T) {
	ctx := context.Background()
	db := openRepositoryTestDatabase(t, ctx)
	if err := audit.Migrate(ctx, db); err != nil {
		t.Fatalf("migrate audit events: %v", err)
	}
	if err := Migrate(ctx, db); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	repository := NewRepository(db, audit.NewWriter())
	createdAt := time.Date(2026, time.August, 29, 10, 0, 0, 0, time.UTC)
	project := Project{
		ID:        uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		Name:      "billing-api",
		CreatedAt: createdAt,
	}
	auditEvent := audit.Event{
		ID:         uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"),
		OccurredAt: createdAt,
		Action:     auditActionProjectCreated,
		Actor:      auditActorInstanceAdmin,
		ProjectID:  project.ID,
		TargetID:   project.ID.String(),
	}

	if err := repository.Create(ctx, project, auditEvent); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	var storedProject projectRecord
	if err := db.WithContext(ctx).First(&storedProject).Error; err != nil {
		t.Fatalf("query stored project: %v", err)
	}
	if storedProject.ID != project.ID.String() || storedProject.Name != project.Name || !storedProject.CreatedAt.Equal(project.CreatedAt) {
		t.Errorf("stored project = %#v, want %#v", storedProject, project)
	}

	var auditEventCount int64
	if err := db.WithContext(ctx).Table("audit_events").Where("id = ?", auditEvent.ID.String()).Count(&auditEventCount).Error; err != nil {
		t.Fatalf("count stored audit events: %v", err)
	}
	if auditEventCount != 1 {
		t.Errorf("stored audit event count = %d, want 1", auditEventCount)
	}
}

func TestRepositoryRollsBackProjectWhenAuditInsertFails(t *testing.T) {
	ctx := context.Background()
	db := openRepositoryTestDatabase(t, ctx)
	if err := audit.Migrate(ctx, db); err != nil {
		t.Fatalf("migrate audit events: %v", err)
	}
	if err := Migrate(ctx, db); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	if err := db.Migrator().DropTable("audit_events"); err != nil {
		t.Fatalf("DropTable(audit_events) error = %v", err)
	}
	repository := NewRepository(db, audit.NewWriter())
	project := Project{
		ID:        uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		Name:      "billing-api",
		CreatedAt: time.Date(2026, time.August, 29, 10, 0, 0, 0, time.UTC),
	}
	auditEvent := audit.Event{
		ID:         uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"),
		OccurredAt: project.CreatedAt,
		Action:     auditActionProjectCreated,
		Actor:      auditActorInstanceAdmin,
		ProjectID:  project.ID,
		TargetID:   project.ID.String(),
	}

	if err := repository.Create(ctx, project, auditEvent); err == nil {
		t.Fatal("Create() error = nil, want audit insert failure")
	}

	var count int64
	if err := db.WithContext(ctx).Model(&projectRecord{}).Count(&count).Error; err != nil {
		t.Fatalf("count stored projects: %v", err)
	}
	if count != 0 {
		t.Errorf("stored project count = %d, want 0 after audit insert failure", count)
	}
}

func TestRepositoryListsProjectsByCreationTimeThenID(t *testing.T) {
	ctx := context.Background()
	db := openRepositoryTestDatabase(t, ctx)
	if err := audit.Migrate(ctx, db); err != nil {
		t.Fatalf("migrate audit events: %v", err)
	}
	if err := Migrate(ctx, db); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	repository := NewRepository(db, audit.NewWriter())
	first := Project{
		ID:        uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		Name:      "billing-api",
		CreatedAt: time.Date(2026, time.August, 29, 10, 0, 0, 0, time.UTC),
	}
	second := Project{
		ID:        uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"),
		Name:      "reporting-api",
		CreatedAt: time.Date(2026, time.August, 29, 11, 0, 0, 0, time.UTC),
	}
	for _, project := range []Project{second, first} {
		auditEvent := audit.Event{
			ID:         uuid.New(),
			OccurredAt: project.CreatedAt,
			Action:     auditActionProjectCreated,
			Actor:      auditActorInstanceAdmin,
			ProjectID:  project.ID,
			TargetID:   project.ID.String(),
		}
		if err := repository.Create(ctx, project, auditEvent); err != nil {
			t.Fatalf("Create(%s) error = %v", project.ID, err)
		}
	}

	got, err := repository.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(got) != 2 || got[0] != first || got[1] != second {
		t.Errorf("List() = %#v, want %#v", got, []Project{first, second})
	}
}

func TestMigrateCanRunMoreThanOnce(t *testing.T) {
	ctx := context.Background()
	db := openRepositoryTestDatabase(t, ctx)

	if err := Migrate(ctx, db); err != nil {
		t.Fatalf("first Migrate() error = %v", err)
	}
	if err := Migrate(ctx, db); err != nil {
		t.Fatalf("second Migrate() error = %v", err)
	}
}

func openRepositoryTestDatabase(t *testing.T, ctx context.Context) *gorm.DB {
	t.Helper()

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
	return connection.DB()
}
