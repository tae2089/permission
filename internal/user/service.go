package user

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/tae2089/trace/v3"

	"github.com/tae2089/go-template/internal/apperr"
)

const maxNameLength = 100

type Service interface {
	Create(context.Context, CreateInput) (User, error)
}

type service struct {
	repository Repository
	newID      func() (uuid.UUID, error)
}

func NewService(repository Repository, newID func() (uuid.UUID, error)) Service {
	return &service{
		repository: repository,
		newID:      newID,
	}
}

func (s *service) Create(ctx context.Context, input CreateInput) (User, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return User{}, apperr.New(apperr.KindBadParameter, "name is required")
	}
	if utf8.RuneCountInString(name) > maxNameLength {
		return User{}, apperr.New(apperr.KindBadParameter, "name must be at most 100 characters")
	}

	email := strings.ToLower(strings.TrimSpace(input.Email))
	if email == "" {
		return User{}, apperr.New(apperr.KindBadParameter, "email is required")
	}

	id, err := s.newID()
	if err != nil {
		return User{}, trace.Wrap(err, "generate user id")
	}
	created := User{
		ID:    id,
		Name:  name,
		Email: email,
	}
	if err := s.repository.Create(ctx, created); err != nil {
		return User{}, trace.Wrap(err, "create user")
	}

	return created, nil
}
