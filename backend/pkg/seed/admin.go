package seed

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/repo/persistent"
	"github.com/skr1ms/CTFBoard/pkg/logger"
	"golang.org/x/crypto/bcrypt"
)

func CreateDefaultAdmin(ctx context.Context, userRepo persistent.UserRepo, username, email, password string, log logger.Logger) error {
	// Check if admin already exists by email
	_, err := userRepo.GetByEmail(ctx, email)
	if err == nil {
		log.Info("Seed: default admin already exists, skipping")
		return nil
	}

	// Create admin user
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	now := time.Now()
	user := &entity.User{
		ID:           uuid.New(),
		TeamID:       nil,
		Username:     username,
		Email:        email,
		PasswordHash: string(hashedPassword),
		Role:         entity.RoleAdmin,
		IsVerified:   true,
		VerifiedAt:   &now,
		CreatedAt:    now,
	}

	if err := userRepo.Create(ctx, user); err != nil {
		return err
	}

	log.Info("Seed: default admin created successfully", map[string]any{"username": username, "email": email})
	return nil
}
