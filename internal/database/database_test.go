package database

import (
	"context"
	"testing"

	"gorm.io/gorm/logger"
)

type testRecord struct {
	ID   uint
	Name string
}

func TestOpenSupportsGORMOperations(t *testing.T) {
	ctx := context.Background()
	connection, err := Open(ctx, Options{
		Driver: DriverSQLite,
		DSN:    "file:database-test?mode=memory&cache=shared",
	})
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	db := connection.DB().WithContext(ctx)
	if err := db.AutoMigrate(&testRecord{}); err != nil {
		t.Fatalf("AutoMigrate() error = %v", err)
	}
	if err := db.Create(&testRecord{Name: "stored"}).Error; err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	var got testRecord
	if err := db.First(&got).Error; err != nil {
		t.Fatalf("First() error = %v", err)
	}
	if got.Name != "stored" {
		t.Errorf("record name = %q, want %q", got.Name, "stored")
	}

	sqlDB, err := connection.DB().DB()
	if err != nil {
		t.Fatalf("DB() error = %v", err)
	}
	if err := connection.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if err := sqlDB.PingContext(ctx); err == nil {
		t.Error("PingContext() error = nil after Close(), want closed database error")
	}
}

func TestOpenDisablesGORMLogging(t *testing.T) {
	connection, err := Open(context.Background(), Options{
		Driver: DriverSQLite,
		DSN:    "file:database-logging-test?mode=memory&cache=shared",
	})
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		if err := connection.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	})

	if connection.DB().Config.Logger != logger.Discard {
		t.Error("GORM logger is enabled, want discarded database logs")
	}
}

func TestOpenRejectsEmptyDSN(t *testing.T) {
	_, err := Open(context.Background(), Options{
		Driver: DriverSQLite,
		DSN:    " ",
	})
	if err == nil {
		t.Fatal("Open() error = nil, want empty DSN error")
	}
}

func TestOpenRejectsUnsupportedDriver(t *testing.T) {
	_, err := Open(context.Background(), Options{
		Driver: "postgres",
		DSN:    "ignored",
	})
	if err == nil {
		t.Fatal("Open() error = nil, want unsupported driver error")
	}
}
