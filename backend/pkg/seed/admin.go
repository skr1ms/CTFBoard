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

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
)

func CreateDefaultAdmin(ctx context.Context, userRepo repo.UserRepository, username, email, password string, log logkit.Logger) error {
	email = strings.ToLower(strings.TrimSpace(email))
	_, err := userRepo.GetByEmail(ctx, email)
	if err == nil {
		log.Info("Seed: default admin already exists, skipping")
		return nil
	}
	if !errors.Is(err, httperr.ErrUserNotFound) {
		return fmt.Errorf("seed admin GetByEmail: %w", err)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("seed admin bcrypt: %w", err)
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
		return fmt.Errorf("seed admin create: %w", err)
	}

	log.Info("Seed: default admin created successfully", map[string]any{"username": username, "email": email})
	return nil
}
