package user

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/tae2089/go-template/internal/apperr"
	"github.com/tae2089/go-template/internal/database"
)

func TestRepositoryPersistsUser(t *testing.T) {
	ctx := context.Background()
	db := openRepositoryTestDatabase(t, ctx)
	if err := Migrate(ctx, db); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	var repository Repository = NewRepository(db)
	want := User{
		ID:    uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		Name:  "Alice",
		Email: "alice@example.com",
	}

	if err := repository.Create(ctx, want); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	var got struct {
		ID    string
		Name  string
		Email string
	}
	if err := db.WithContext(ctx).Table("users").First(&got).Error; err != nil {
		t.Fatalf("query stored user: %v", err)
	}
	if got.ID != want.ID.String() || got.Name != want.Name || got.Email != want.Email {
		t.Errorf("stored user = %#v, want %#v", got, want)
	}
}

func TestRepositoryReturnsAlreadyExistsForDuplicateEmail(t *testing.T) {
	ctx := context.Background()
	db := openRepositoryTestDatabase(t, ctx)
	if err := Migrate(ctx, db); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	repository := NewRepository(db)
	first := User{
		ID:    uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		Name:  "Alice",
		Email: "alice@example.com",
	}
	if err := repository.Create(ctx, first); err != nil {
		t.Fatalf("first Create() error = %v", err)
	}

	err := repository.Create(ctx, User{
		ID:    uuid.MustParse("550e8400-e29b-41d4-a716-446655440001"),
		Name:  "Another Alice",
		Email: first.Email,
	})

	var target *apperr.Error
	if !errors.As(err, &target) || target.Kind != apperr.KindAlreadyExists {
		t.Errorf("duplicate Create() error = %v, want already exists", err)
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
