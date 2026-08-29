package user

import (
	"context"
	"errors"

	"github.com/tae2089/trace/v3"
	"gorm.io/gorm"

	"github.com/tae2089/go-template/internal/apperr"
)

type Repository interface {
	Create(context.Context, User) error
}

type repository struct {
	db *gorm.DB
}

type record struct {
	ID    string `gorm:"primaryKey;size:36"`
	Name  string `gorm:"not null"`
	Email string `gorm:"not null;uniqueIndex"`
}

func (record) TableName() string {
	return "users"
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, user User) error {
	err := r.db.WithContext(ctx).Create(&record{
		ID:    user.ID.String(),
		Name:  user.Name,
		Email: user.Email,
	}).Error
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return apperr.Wrap(apperr.KindAlreadyExists, err, "user already exists")
	}
	if err != nil {
		return trace.Wrap(err, "insert user")
	}
	return nil
}
