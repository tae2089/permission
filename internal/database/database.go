package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Connection struct {
	db    *gorm.DB
	sqlDB *sql.DB
}

type Driver string

const DriverSQLite Driver = "sqlite"

type Options struct {
	Driver Driver
	DSN    string
}

func Open(ctx context.Context, opts Options) (*Connection, error) {
	var dialector gorm.Dialector
	switch opts.Driver {
	case DriverSQLite:
		dialector = sqlite.Open(opts.DSN)
	default:
		return nil, fmt.Errorf("unsupported database driver %q", opts.Driver)
	}

	if strings.TrimSpace(opts.DSN) == "" {
		return nil, errors.New("database dsn is required")
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		DisableAutomaticPing: true,
		Logger:               logger.Discard,
		TranslateError:       true,
	})
	if err != nil {
		return nil, fmt.Errorf("open %s database: %w", opts.Driver, err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get database connection pool: %w", err)
	}
	if err := sqlDB.PingContext(ctx); err != nil {
		pingErr := fmt.Errorf("ping database: %w", err)
		if closeErr := sqlDB.Close(); closeErr != nil {
			return nil, errors.Join(pingErr, fmt.Errorf("close database after ping failure: %w", closeErr))
		}
		return nil, pingErr
	}

	return &Connection{
		db:    db,
		sqlDB: sqlDB,
	}, nil
}

func (c *Connection) DB() *gorm.DB {
	return c.db
}

func (c *Connection) Close() error {
	if err := c.sqlDB.Close(); err != nil {
		return fmt.Errorf("close database: %w", err)
	}
	return nil
}
