package project

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/tae2089/go-template/internal/audit"
	"github.com/tae2089/trace/v3"
	"gorm.io/gorm"
)

type Repository interface {
	Create(context.Context, Project, audit.Event) error
	List(context.Context) ([]Project, error)
}

type repository struct {
	db          *gorm.DB
	auditWriter *audit.Writer
}

type projectRecord struct {
	ID        string    `gorm:"primaryKey;size:36"`
	Name      string    `gorm:"not null"`
	CreatedAt time.Time `gorm:"not null"`
}

func (projectRecord) TableName() string {
	return "projects"
}

func NewRepository(db *gorm.DB, auditWriter *audit.Writer) Repository {
	return &repository{
		db:          db,
		auditWriter: auditWriter,
	}
}

func (r *repository) Create(ctx context.Context, project Project, auditEvent audit.Event) error {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&projectRecord{
			ID:        project.ID.String(),
			Name:      project.Name,
			CreatedAt: project.CreatedAt,
		}).Error; err != nil {
			return trace.Wrap(err, "insert project")
		}
		if err := r.auditWriter.Append(ctx, tx, auditEvent); err != nil {
			return trace.Wrap(err, "append project audit event")
		}
		return nil
	})
	if err != nil {
		return trace.Wrap(err, "create project with audit event")
	}
	return nil
}

func (r *repository) List(ctx context.Context) ([]Project, error) {
	var records []projectRecord
	if err := r.db.WithContext(ctx).
		Order("created_at ASC").
		Order("id ASC").
		Find(&records).Error; err != nil {
		return nil, trace.Wrap(err, "query projects")
	}

	projects := make([]Project, 0, len(records))
	for _, record := range records {
		id, err := uuid.Parse(record.ID)
		if err != nil {
			return nil, trace.Wrap(err, "parse stored project id")
		}
		projects = append(projects, Project{
			ID:        id,
			Name:      record.Name,
			CreatedAt: record.CreatedAt,
		})
	}
	return projects, nil
}
