package auth

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// UserServiceAdapter implements the waitlist UserService interface using the auth repository
// This adapter prevents import cycles while allowing waitlist service to access user data
type UserServiceAdapter struct {
	repo Repository
}

// NewUserServiceAdapter creates a new user service adapter
func NewUserServiceAdapter(repo Repository) *UserServiceAdapter {
	return &UserServiceAdapter{
		repo: repo,
	}
}

// GetUserByID fetches user details by ID and returns email, firstName, lastName
// This implements the UserService interface expected by the waitlist service
func (usa *UserServiceAdapter) GetUserByID(ctx context.Context, userID uuid.UUID) (email, firstName, lastName string, err error) {
	// Convert UUID to string as the repository expects string
	userIDStr := userID.String()

	user, err := usa.repo.GetUserByID(ctx, userIDStr)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to fetch user %s: %w", userID, err)
	}

	return user.Email, user.FirstName, user.LastName, nil
}
