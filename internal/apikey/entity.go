package apikey

import (
	"time"

	"github.com/google/uuid"
)

type Kind string

const (
	KindProjectAdmin Kind = "project_admin"
	KindDecision     Kind = "decision"
)

type Status string

const (
	StatusActive  Status = "active"
	StatusRevoked Status = "revoked"
)

type Key struct {
	ID         uuid.UUID
	ProjectID  uuid.UUID
	Kind       Kind
	Status     Status
	SecretHash string
	CreatedAt  time.Time
	RevokedAt  *time.Time
}

type IssuedKey struct {
	Key    Key
	Secret string
}

type IssueInput struct {
	ProjectID uuid.UUID
	Kind      Kind
}

type InstanceAdministrator struct{}

type ProjectAdministrator struct {
	ProjectID uuid.UUID
	KeyID     uuid.UUID
}
