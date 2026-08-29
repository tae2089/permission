package audit

import (
	"context"
	"time"

	"github.com/tae2089/trace/v3"
	"gorm.io/gorm"
)

type Writer struct{}

type record struct {
	ID         string    `gorm:"primaryKey;size:36"`
	OccurredAt time.Time `gorm:"not null"`
	Action     string    `gorm:"not null"`
	Actor      string    `gorm:"not null"`
	ProjectID  string    `gorm:"not null;index"`
	TargetID   string    `gorm:"not null"`
}

func (record) TableName() string {
	return "audit_events"
}

func NewWriter() *Writer {
	return &Writer{}
}

func (w *Writer) Append(ctx context.Context, tx *gorm.DB, event Event) error {
	if err := tx.WithContext(ctx).Create(&record{
		ID:         event.ID.String(),
		OccurredAt: event.OccurredAt,
		Action:     event.Action,
		Actor:      event.Actor,
		ProjectID:  event.ProjectID.String(),
		TargetID:   event.TargetID,
	}).Error; err != nil {
		return trace.Wrap(err, "insert audit event")
	}
	return nil
}
