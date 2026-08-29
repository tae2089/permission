package apikey

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
	"github.com/tae2089/go-template/internal/apperr"
	"github.com/tae2089/go-template/internal/audit"
	"github.com/tae2089/trace/v3"
)

const (
	auditActionKeyIssued    = "api_key.issued"
	auditActionKeyRotated   = "api_key.rotated"
	auditActionKeyRevoked   = "api_key.revoked"
	auditActorInstanceAdmin = "instance_admin"
)

type Repository interface {
	Create(context.Context, Key, audit.Event) error
	Get(context.Context, uuid.UUID, uuid.UUID) (Key, error)
	FindActiveBySecretHash(context.Context, string) (Key, error)
	List(context.Context, uuid.UUID) ([]Key, error)
	Revoke(context.Context, uuid.UUID, uuid.UUID, audit.Event) error
}

type Service interface {
	Issue(context.Context, any, IssueInput) (IssuedKey, error)
	Authenticate(context.Context, string) (Key, error)
	List(context.Context, any, uuid.UUID) ([]Key, error)
	Rotate(context.Context, ProjectAdministrator, uuid.UUID, uuid.UUID) (IssuedKey, error)
	Revoke(context.Context, ProjectAdministrator, uuid.UUID, uuid.UUID) error
}

type service struct {
	repository Repository
}

func NewService(repository Repository) Service {
	return &service{repository: repository}
}

func (s *service) Issue(ctx context.Context, actor any, input IssueInput) (IssuedKey, error) {
	if input.ProjectID == uuid.Nil || (input.Kind != KindProjectAdmin && input.Kind != KindDecision) {
		return IssuedKey{}, apperr.New(apperr.KindBadParameter, "invalid API key issue request")
	}
	if !canIssue(actor, input.ProjectID, input.Kind) {
		return IssuedKey{}, apperr.New(apperr.KindAccessDenied, "")
	}
	return s.issue(ctx, actor, input, auditActionKeyIssued)
}

func (s *service) Authenticate(ctx context.Context, secret string) (Key, error) {
	if secret == "" {
		return Key{}, apperr.New(apperr.KindUnauthenticated, "")
	}
	digest := sha256.Sum256([]byte(secret))
	key, err := s.repository.FindActiveBySecretHash(ctx, hex.EncodeToString(digest[:]))
	if err != nil {
		return Key{}, trace.Wrap(err, "authenticate API key")
	}
	return key, nil
}

func (s *service) List(ctx context.Context, actor any, projectID uuid.UUID) ([]Key, error) {
	if !canManage(actor, projectID) {
		return nil, apperr.New(apperr.KindAccessDenied, "")
	}
	keys, err := s.repository.List(ctx, projectID)
	if err != nil {
		return nil, trace.Wrap(err, "list API keys")
	}
	return keys, nil
}

func (s *service) Rotate(ctx context.Context, actor ProjectAdministrator, projectID, keyID uuid.UUID) (IssuedKey, error) {
	if actor.ProjectID != projectID {
		return IssuedKey{}, apperr.New(apperr.KindAccessDenied, "")
	}
	key, err := s.repository.Get(ctx, projectID, keyID)
	if err != nil {
		return IssuedKey{}, trace.Wrap(err, "get API key to rotate")
	}
	if key.Status != StatusActive {
		return IssuedKey{}, apperr.New(apperr.KindConflict, "API key is not active")
	}
	return s.issue(ctx, actor, IssueInput{ProjectID: projectID, Kind: key.Kind}, auditActionKeyRotated)
}

func (s *service) Revoke(ctx context.Context, actor ProjectAdministrator, projectID, keyID uuid.UUID) error {
	if actor.ProjectID != projectID {
		return apperr.New(apperr.KindAccessDenied, "")
	}
	eventID, err := uuid.NewRandom()
	if err != nil {
		return trace.Wrap(err, "generate API key audit event id")
	}
	now := time.Now().UTC()
	event := audit.Event{ID: eventID, OccurredAt: now, Action: auditActionKeyRevoked, Actor: auditActor(actor), ProjectID: projectID, TargetID: keyID.String()}
	if err := s.repository.Revoke(ctx, projectID, keyID, event); err != nil {
		return trace.Wrap(err, "revoke API key")
	}
	return nil
}

func (s *service) issue(ctx context.Context, actor any, input IssueInput, action string) (IssuedKey, error) {
	secret, err := NewSecret()
	if err != nil {
		return IssuedKey{}, trace.Wrap(err, "generate API key secret")
	}
	id, err := uuid.NewRandom()
	if err != nil {
		return IssuedKey{}, trace.Wrap(err, "generate API key id")
	}
	now := time.Now().UTC()
	digest := sha256.Sum256([]byte(secret))
	key := Key{ID: id, ProjectID: input.ProjectID, Kind: input.Kind, Status: StatusActive, SecretHash: hex.EncodeToString(digest[:]), CreatedAt: now}
	auditID, err := uuid.NewRandom()
	if err != nil {
		return IssuedKey{}, trace.Wrap(err, "generate API key audit event id")
	}
	event := audit.Event{ID: auditID, OccurredAt: now, Action: action, Actor: auditActor(actor), ProjectID: input.ProjectID, TargetID: id.String()}
	if err := s.repository.Create(ctx, key, event); err != nil {
		return IssuedKey{}, trace.Wrap(err, "issue API key")
	}
	return IssuedKey{Key: key, Secret: secret}, nil
}

func canIssue(actor any, projectID uuid.UUID, kind Kind) bool {
	switch actor := actor.(type) {
	case InstanceAdministrator:
		return kind == KindProjectAdmin
	case ProjectAdministrator:
		return actor.ProjectID == projectID
	default:
		return false
	}
}

func canManage(actor any, projectID uuid.UUID) bool {
	switch actor := actor.(type) {
	case InstanceAdministrator:
		return true
	case ProjectAdministrator:
		return actor.ProjectID == projectID
	default:
		return false
	}
}

func auditActor(actor any) string {
	if _, ok := actor.(InstanceAdministrator); ok {
		return auditActorInstanceAdmin
	}
	return "project_admin"
}
