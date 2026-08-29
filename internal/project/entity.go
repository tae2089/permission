package project

import (
	"time"

	"github.com/google/uuid"
)

type Project struct {
	ID        uuid.UUID
	Name      string
	CreatedAt time.Time
}

type CreateInput struct {
	Name string
}
