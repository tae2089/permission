package apikey

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/tae2089/go-template/internal/apperr"
	"github.com/tae2089/go-template/internal/audit"
	"github.com/tae2089/trace/v3"
	"gorm.io/gorm"
)

const activeKeyLimit = 2

type repository struct {
	db          *gorm.DB
	auditWriter *audit.Writer
}

type record struct {
	ID         string    `gorm:"primaryKey;size:36"`
	ProjectID  string    `gorm:"not null;index"`
	Kind       string    `gorm:"not null;index"`
	Status     string    `gorm:"not null;index"`
	SecretHash string    `gorm:"not null;uniqueIndex"`
	CreatedAt  time.Time `gorm:"not null"`
	RevokedAt  *time.Time
}

func (record) TableName() string { return "api_keys" }

func NewRepository(db *gorm.DB, auditWriter *audit.Writer) Repository {
	return &repository{db: db, auditWriter: auditWriter}
}

func (r *repository) Create(ctx context.Context, key Key, event audit.Event) error {
	if err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var projects int64
		if err := tx.Table("projects").Where("id = ?", key.ProjectID.String()).Count(&projects).Error; err != nil {
			return trace.Wrap(err, "find API key project")
		}
		if projects == 0 {
			return apperr.New(apperr.KindNotFound, "project not found")
		}
		var active int64
		if err := tx.Model(&record{}).Where("project_id = ? AND kind = ? AND status = ?", key.ProjectID.String(), string(key.Kind), string(StatusActive)).Count(&active).Error; err != nil {
			return trace.Wrap(err, "count active API keys")
		}
		if active >= activeKeyLimit {
			return apperr.New(apperr.KindLimitExceeded, "at most two active API keys of each kind are allowed")
		}
		if err := tx.Create(&record{ID: key.ID.String(), ProjectID: key.ProjectID.String(), Kind: string(key.Kind), Status: string(key.Status), SecretHash: key.SecretHash, CreatedAt: key.CreatedAt}).Error; err != nil {
			return trace.Wrap(err, "insert API key")
		}
		return r.auditWriter.Append(ctx, tx, event)
	}); err != nil {
		return trace.Wrap(err, "create API key with audit event")
	}
	return nil
}

func (r *repository) Get(ctx context.Context, projectID, keyID uuid.UUID) (Key, error) {
	var stored record
	if err := r.db.WithContext(ctx).Where("project_id = ? AND id = ?", projectID.String(), keyID.String()).Take(&stored).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Key{}, apperr.New(apperr.KindNotFound, "API key not found")
		}
		return Key{}, trace.Wrap(err, "query API key")
	}
	return stored.key()
}

func (r *repository) FindActiveBySecretHash(ctx context.Context, hash string) (Key, error) {
	var stored record
	if err := r.db.WithContext(ctx).Where("secret_hash = ? AND status = ?", hash, string(StatusActive)).Take(&stored).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Key{}, apperr.New(apperr.KindUnauthenticated, "")
		}
		return Key{}, trace.Wrap(err, "query active API key")
	}
	return stored.key()
}

func (r *repository) List(ctx context.Context, projectID uuid.UUID) ([]Key, error) {
	var records []record
	if err := r.db.WithContext(ctx).Where("project_id = ?", projectID.String()).Order("created_at ASC").Order("id ASC").Find(&records).Error; err != nil {
		return nil, trace.Wrap(err, "query API keys")
	}
	keys := make([]Key, 0, len(records))
	for _, stored := range records {
		key, err := stored.key()
		if err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	return keys, nil
}

func (r *repository) Revoke(ctx context.Context, projectID, keyID uuid.UUID, event audit.Event) error {
	now := event.OccurredAt
	if err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&record{}).Where("project_id = ? AND id = ? AND status = ?", projectID.String(), keyID.String(), string(StatusActive)).Updates(map[string]any{"status": string(StatusRevoked), "revoked_at": now})
		if result.Error != nil {
			return trace.Wrap(result.Error, "revoke API key")
		}
		if result.RowsAffected == 0 {
			return apperr.New(apperr.KindNotFound, "active API key not found")
		}
		return r.auditWriter.Append(ctx, tx, event)
	}); err != nil {
		return trace.Wrap(err, "revoke API key with audit event")
	}
	return nil
}

func (r record) key() (Key, error) {
	id, err := uuid.Parse(r.ID)
	if err != nil {
		return Key{}, trace.Wrap(err, "parse stored API key id")
	}
	projectID, err := uuid.Parse(r.ProjectID)
	if err != nil {
		return Key{}, trace.Wrap(err, "parse stored API key project id")
	}
	return Key{ID: id, ProjectID: projectID, Kind: Kind(r.Kind), Status: Status(r.Status), SecretHash: r.SecretHash, CreatedAt: r.CreatedAt, RevokedAt: r.RevokedAt}, nil
}
