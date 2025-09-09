// internal/auth/repository.go
package auth

import (
	"context"
	"errors"

	"evently/internal/users"

	"gorm.io/gorm"
)

type Repository interface {
	CreateUser(ctx context.Context, user *users.User) error
	GetUserByEmail(ctx context.Context, email string) (*users.User, error)
	GetUserByID(ctx context.Context, id string) (*users.User, error)
	UpdateUserPassword(ctx context.Context, userID string, hashedPassword string) error
	EmailExists(ctx context.Context, email string) (bool, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{
		db: db,
	}
}

func (r *repository) CreateUser(ctx context.Context, user *users.User) error {
	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		return err
	}
	return nil
}

func (r *repository) GetUserByEmail(ctx context.Context, email string) (*users.User, error) {
	var user users.User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *repository) GetUserByID(ctx context.Context, id string) (*users.User, error) {
	var user users.User
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *repository) UpdateUserPassword(ctx context.Context, userID string, hashedPassword string) error {
	result := r.db.WithContext(ctx).Model(&users.User{}).
		Where("id = ?", userID).
		Update("password", hashedPassword)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}

	return nil
}

func (r *repository) EmailExists(ctx context.Context, email string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&users.User{}).Where("email = ?", email).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
