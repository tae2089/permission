package apikey

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/tae2089/go-template/internal/apperr"
	"github.com/tae2089/go-template/internal/audit"
)

func TestServiceIssuesProjectAdminKeyForInstanceAdministrator(t *testing.T) {
	projectID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	var stored Key
	var storedEvent audit.Event
	service := NewService(repositoryFunc(func(_ context.Context, key Key, event audit.Event) error {
		stored = key
		storedEvent = event
		return nil
	}))

	issued, err := service.Issue(context.Background(), InstanceAdministrator{}, IssueInput{ProjectID: projectID, Kind: KindProjectAdmin})
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	if issued.Secret == "" {
		t.Error("Secret is empty")
	}
	if stored.ID == uuid.Nil || stored.ID != issued.Key.ID || stored.ProjectID != projectID || stored.Kind != KindProjectAdmin || stored.Status != StatusActive || stored.CreatedAt.IsZero() {
		t.Errorf("stored key = %#v, want active project administrator key", stored)
	}
	if stored.SecretHash == issued.Secret || stored.SecretHash == "" {
		t.Error("stored key does not contain a derived secret hash")
	}
	if storedEvent.ID == uuid.Nil || storedEvent.OccurredAt.IsZero() || !storedEvent.OccurredAt.Equal(stored.CreatedAt) || storedEvent.Action != auditActionKeyIssued || storedEvent.Actor != auditActorInstanceAdmin || storedEvent.ProjectID != projectID || storedEvent.TargetID != issued.Key.ID.String() {
		t.Errorf("stored audit event = %#v, want project-admin key issue event", storedEvent)
	}
}

func TestServiceRejectsCrossProjectProjectAdministrator(t *testing.T) {
	service := NewService(repositoryFunc(func(context.Context, Key, audit.Event) error {
		t.Fatal("repository called")
		return nil
	}))

	_, err := service.Issue(context.Background(), ProjectAdministrator{ProjectID: uuid.New(), KeyID: uuid.New()}, IssueInput{ProjectID: uuid.New(), Kind: KindDecision})

	var target *apperr.Error
	if !errors.As(err, &target) || target.Kind != apperr.KindAccessDenied {
		t.Errorf("Issue() error = %v, want access denied", err)
	}
}

func TestServiceRejectsInstanceAdministratorDecisionKeyIssue(t *testing.T) {
	service := NewService(repositoryFunc(func(context.Context, Key, audit.Event) error {
		t.Fatal("repository called")
		return nil
	}))

	_, err := service.Issue(context.Background(), InstanceAdministrator{}, IssueInput{ProjectID: uuid.New(), Kind: KindDecision})

	var target *apperr.Error
	if !errors.As(err, &target) || target.Kind != apperr.KindAccessDenied {
		t.Errorf("Issue() error = %v, want access denied", err)
	}
}

type repositoryFunc func(context.Context, Key, audit.Event) error

func (f repositoryFunc) Create(ctx context.Context, key Key, event audit.Event) error {
	return f(ctx, key, event)
}

func (repositoryFunc) Get(context.Context, uuid.UUID, uuid.UUID) (Key, error) {
	return Key{}, nil
}

func (repositoryFunc) FindActiveBySecretHash(context.Context, string) (Key, error) {
	return Key{}, nil
}

func (repositoryFunc) List(context.Context, uuid.UUID) ([]Key, error) {
	return nil, nil
}

func (repositoryFunc) Revoke(context.Context, uuid.UUID, uuid.UUID, audit.Event) error {
	return nil
}
