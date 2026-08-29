package user

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/tae2089/go-template/internal/apperr"
)

func TestServiceCreatesNormalizedUser(t *testing.T) {
	userID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	var stored User
	var service Service = NewService(repositoryFunc(func(_ context.Context, user User) error {
		stored = user
		return nil
	}), fixedID(userID))

	created, err := service.Create(context.Background(), CreateInput{
		Name:  " Alice ",
		Email: " ALICE@Example.COM ",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.ID != userID {
		t.Errorf("created ID = %s, want %s", created.ID, userID)
	}
	if created.Name != "Alice" {
		t.Errorf("created name = %q, want %q", created.Name, "Alice")
	}
	if created.Email != "alice@example.com" {
		t.Errorf("created email = %q, want %q", created.Email, "alice@example.com")
	}
	if stored != created {
		t.Errorf("stored user = %#v, want %#v", stored, created)
	}
}

func TestServiceRejectsInvalidUser(t *testing.T) {
	tests := []struct {
		name  string
		input CreateInput
	}{
		{name: "empty name", input: CreateInput{Name: " ", Email: "alice@example.com"}},
		{name: "name over 100 characters", input: CreateInput{Name: strings.Repeat("가", 101), Email: "alice@example.com"}},
		{name: "empty email", input: CreateInput{Name: "Alice", Email: " "}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repositoryCalled := false
			idGeneratorCalled := false
			service := NewService(repositoryFunc(func(context.Context, User) error {
				repositoryCalled = true
				return nil
			}), func() (uuid.UUID, error) {
				idGeneratorCalled = true
				return uuid.New(), nil
			})

			_, err := service.Create(context.Background(), tt.input)

			var target *apperr.Error
			if !errors.As(err, &target) || target.Kind != apperr.KindBadParameter {
				t.Errorf("Create() error = %v, want bad parameter", err)
			}
			if repositoryCalled {
				t.Error("repository was called for invalid user")
			}
			if idGeneratorCalled {
				t.Error("ID generator was called for invalid user")
			}
		})
	}
}

func TestServicePreservesRepositoryErrorMeaning(t *testing.T) {
	repositoryErr := apperr.New(apperr.KindAlreadyExists, "user already exists")
	service := NewService(repositoryFunc(func(context.Context, User) error {
		return repositoryErr
	}), fixedID(uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")))

	_, err := service.Create(context.Background(), CreateInput{
		Name:  "Alice",
		Email: "alice@example.com",
	})

	var target *apperr.Error
	if !errors.As(err, &target) || target.Kind != apperr.KindAlreadyExists {
		t.Errorf("Create() error = %v, want already exists", err)
	}
	if !errors.Is(err, repositoryErr) {
		t.Errorf("Create() error does not preserve repository cause: %v", err)
	}
}

func TestServicePreservesIDGenerationError(t *testing.T) {
	idErr := errors.New("random source failed")
	repositoryCalled := false
	service := NewService(repositoryFunc(func(context.Context, User) error {
		repositoryCalled = true
		return nil
	}), func() (uuid.UUID, error) {
		return uuid.Nil, idErr
	})

	_, err := service.Create(context.Background(), CreateInput{
		Name:  "Alice",
		Email: "alice@example.com",
	})

	if !errors.Is(err, idErr) {
		t.Errorf("Create() error = %v, want ID generation cause", err)
	}
	if repositoryCalled {
		t.Error("repository was called after ID generation failed")
	}
}

type repositoryFunc func(context.Context, User) error

func (f repositoryFunc) Create(ctx context.Context, user User) error {
	return f(ctx, user)
}

func fixedID(id uuid.UUID) func() (uuid.UUID, error) {
	return func() (uuid.UUID, error) {
		return id, nil
	}
}
