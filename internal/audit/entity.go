package audit

import (
	"time"

	"github.com/google/uuid"
)

type Event struct {
	ID         uuid.UUID
	OccurredAt time.Time
	Action     string
	Actor      string
	ProjectID  uuid.UUID
	TargetID   string
}
