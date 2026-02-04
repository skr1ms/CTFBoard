package user

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/repo"
)

type APITokenUseCase struct {
	repo repo.APITokenRepository
}

func NewAPITokenUseCase(
	repo repo.APITokenRepository,
) *APITokenUseCase {
	return &APITokenUseCase{repo: repo}
}

func (uc *APITokenUseCase) List(ctx context.Context, userID uuid.UUID) ([]*entity.APIToken, error) {
	return uc.repo.GetByUserID(ctx, userID)
}

func (uc *APITokenUseCase) Create(ctx context.Context, userID uuid.UUID, description string, expiresAt *time.Time) (plaintext string, token *entity.APIToken, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", nil, fmt.Errorf("APITokenUseCase - Create - rand: %w", err)
	}
	plaintext = hex.EncodeToString(b)
	hash := sha256.Sum256([]byte(plaintext))
	tokenHash := hex.EncodeToString(hash[:])

	token = &entity.APIToken{
		UserID:      userID,
		TokenHash:   tokenHash,
		Description: description,
		ExpiresAt:   expiresAt,
		CreatedAt:   time.Now(),
	}
	if err := uc.repo.Create(ctx, token); err != nil {
		return "", nil, fmt.Errorf("APITokenUseCase - Create: %w", err)
	}
	return plaintext, token, nil
}

func (uc *APITokenUseCase) Delete(ctx context.Context, id, userID uuid.UUID) error {
	return uc.repo.Delete(ctx, id, userID)
}

func (uc *APITokenUseCase) GetByTokenHash(ctx context.Context, tokenHash string) (*entity.APIToken, error) {
	return uc.repo.GetByTokenHash(ctx, tokenHash)
}

func (uc *APITokenUseCase) UpdateLastUsedAt(ctx context.Context, id uuid.UUID) error {
	return uc.repo.UpdateLastUsedAt(ctx, id, time.Now())
}

func (uc *APITokenUseCase) ValidateToken(t *entity.APIToken) bool {
	if t == nil {
		return false
	}
	if t.ExpiresAt != nil && t.ExpiresAt.Before(time.Now()) {
		return false
	}
	return true
}
