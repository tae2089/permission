package project

import (
	"context"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/tae2089/go-template/internal/apperr"
	"github.com/tae2089/go-template/internal/audit"
	"github.com/tae2089/trace/v3"
)

var projectNamePattern = regexp.MustCompile(`^[a-z0-9-]{1,63}$`)

type Service interface {
	Create(context.Context, CreateInput) (Project, error)
	List(context.Context) ([]Project, error)
}

type service struct {
	repository Repository
	newID      func() (uuid.UUID, error)
	now        func() time.Time
}

const (
	auditActionProjectCreated = "project.created"
	auditActorInstanceAdmin   = "instance_admin"
)

func NewService(
	repository Repository,
	newID func() (uuid.UUID, error),
	now func() time.Time,
) Service {
	return &service{
		repository: repository,
		newID:      newID,
		now:        now,
	}
}

func (s *service) Create(ctx context.Context, input CreateInput) (Project, error) {
	if !projectNamePattern.MatchString(input.Name) {
		return Project{}, apperr.New(
			apperr.KindBadParameter,
			"project name must contain 1 to 63 lowercase letters, digits, or hyphens",
		)
	}

	id, err := s.newID()
	if err != nil {
		return Project{}, trace.Wrap(err, "generate project id")
	}
	created := Project{
		ID:        id,
		Name:      input.Name,
		CreatedAt: s.now().UTC(),
	}
	auditEventID, err := s.newID()
	if err != nil {
		return Project{}, trace.Wrap(err, "generate audit event id")
	}
	auditEvent := audit.Event{
		ID:         auditEventID,
		OccurredAt: created.CreatedAt,
		Action:     auditActionProjectCreated,
		Actor:      auditActorInstanceAdmin,
		ProjectID:  created.ID,
		TargetID:   created.ID.String(),
	}
	if err := s.repository.Create(ctx, created, auditEvent); err != nil {
		return Project{}, trace.Wrap(err, "create project")
	}

	return created, nil
}

func (s *service) List(ctx context.Context) ([]Project, error) {
	projects, err := s.repository.List(ctx)
	if err != nil {
		return nil, trace.Wrap(err, "list projects")
	}
	return projects, nil
}
