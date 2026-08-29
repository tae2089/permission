package audit

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/tae2089/go-template/internal/database"
)

func TestWriterAppendsEvent(t *testing.T) {
	ctx := context.Background()
	db := openWriterTestDatabase(t, ctx)
	if err := Migrate(ctx, db); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	writer := NewWriter()
	event := Event{
		ID:         uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"),
		OccurredAt: time.Date(2026, time.August, 29, 10, 0, 0, 0, time.UTC),
		Action:     "project.created",
		Actor:      "instance_admin",
		ProjectID:  uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		TargetID:   "550e8400-e29b-41d4-a716-446655440000",
	}

	if err := writer.Append(ctx, db, event); err != nil {
		t.Fatalf("Append() error = %v", err)
	}

	var stored struct {
		ID         string
		OccurredAt time.Time
		Action     string
		Actor      string
		ProjectID  string
		TargetID   string
	}
	if err := db.WithContext(ctx).Table("audit_events").First(&stored).Error; err != nil {
		t.Fatalf("query stored audit event: %v", err)
	}
	if stored.ID != event.ID.String() ||
		!stored.OccurredAt.Equal(event.OccurredAt) ||
		stored.Action != event.Action ||
		stored.Actor != event.Actor ||
		stored.ProjectID != event.ProjectID.String() ||
		stored.TargetID != event.TargetID {
		t.Errorf("stored audit event = %#v, want %#v", stored, event)
	}
}

func TestMigrateCanRunMoreThanOnce(t *testing.T) {
	ctx := context.Background()
	db := openWriterTestDatabase(t, ctx)

	if err := Migrate(ctx, db); err != nil {
		t.Fatalf("first Migrate() error = %v", err)
	}
	if err := Migrate(ctx, db); err != nil {
		t.Fatalf("second Migrate() error = %v", err)
	}
}

func openWriterTestDatabase(t *testing.T, ctx context.Context) *gorm.DB {
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
