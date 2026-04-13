package seed

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/wahrwelt-kit/go-logkit"
	"golang.org/x/crypto/bcrypt"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
)

// CreateDefaultAdmin creates an admin user with the given credentials if one with that email does not
// already exist. It is intended to be called once at application startup to ensure a bootstrapped admin account.
// bcryptCost controls the work factor; pass 0 to use bcrypt.DefaultCost (the production default).
func CreateDefaultAdmin(ctx context.Context, userRepo repo.UserRepository, username, email, password string, log logkit.Logger, bcryptCost int) error {
	email = strings.ToLower(strings.TrimSpace(email))

	_, err := userRepo.GetByEmail(ctx, email)
	if err == nil {
		log.Info("Seed: default admin already exists, skipping")

		return nil
	}

	if !errors.Is(err, apperr.ErrUserNotFound) {
		return fmt.Errorf("CreateDefaultAdmin - GetByEmail: %w", err)
	}

	cost := bcryptCost
	if cost == 0 {
		cost = bcrypt.DefaultCost
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return fmt.Errorf("CreateDefaultAdmin - GenerateFromPassword: %w", err)
	}

	now := time.Now()
	user := &domain.User{
		ID:           uuid.New(),
		TeamID:       nil,
		Username:     username,
		Email:        email,
		PasswordHash: string(hashedPassword),
		Role:         domain.RoleAdmin,
		IsVerified:   true,
		VerifiedAt:   &now,
		CreatedAt:    now,
	}

	if err := userRepo.Create(ctx, user); err != nil {
		return fmt.Errorf("CreateDefaultAdmin - Create: %w", err)
	}

	log.Info("Seed: default admin created successfully", map[string]any{"username": username, "email": email})

	return nil
}
