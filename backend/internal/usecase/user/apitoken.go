package user

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/crypto"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
)

const (
	apiTokenDescriptionMaxLen = 255
	apiTokenRandomBytes       = 32
)

type APITokenUseCase struct {
	deps APITokenDeps
}

type APITokenDeps struct {
	Repo repo.APITokenRepository
}

var _ usecase.APITokenUseCase = (*APITokenUseCase)(nil)

func NewAPITokenUseCase(deps APITokenDeps) *APITokenUseCase {
	return &APITokenUseCase{deps: deps}
}

func (uc *APITokenUseCase) List(ctx context.Context, userID uuid.UUID) ([]*domain.APIToken, error) {
	tokens, err := uc.deps.Repo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("APITokenUseCase - List - APITokenRepo.GetByUserID: %w", err)
	}
	return tokens, nil
}

func (uc *APITokenUseCase) Create(ctx context.Context, userID uuid.UUID, description string, expiresAt *time.Time) (plaintext string, token *domain.APIToken, err error) {
	if len(description) > apiTokenDescriptionMaxLen {
		return "", nil, httperr.NewValidationErrorf("description must not exceed %d characters", apiTokenDescriptionMaxLen)
	}
	plaintext, err = crypto.SecureRandomHex(apiTokenRandomBytes)
	if err != nil {
		return "", nil, fmt.Errorf("APITokenUseCase - Create - SecureRandomHex: %w", err)
	}
	tokenHash := crypto.SHA256Hex(plaintext)

	token = &domain.APIToken{
		UserID:      userID,
		TokenHash:   tokenHash,
		Description: description,
		ExpiresAt:   expiresAt,
		CreatedAt:   time.Now(),
	}
	if err := uc.deps.Repo.Create(ctx, token); err != nil {
		return "", nil, fmt.Errorf("APITokenUseCase - Create - APITokenRepo.Create: %w", err)
	}
	return plaintext, token, nil
}

func (uc *APITokenUseCase) Delete(ctx context.Context, ID, userID uuid.UUID) error {
	if err := uc.deps.Repo.Delete(ctx, ID, userID); err != nil {
		return fmt.Errorf("APITokenUseCase - Delete - APITokenRepo.Delete: %w", err)
	}
	return nil
}

func (uc *APITokenUseCase) GetByTokenHash(ctx context.Context, tokenHash string) (*domain.APIToken, error) {
	token, err := uc.deps.Repo.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		return nil, fmt.Errorf("APITokenUseCase - GetByTokenHash - APITokenRepo.GetByTokenHash: %w", err)
	}
	return token, nil
}

func (uc *APITokenUseCase) UpdateLastUsedAt(ctx context.Context, ID uuid.UUID) error {
	if err := uc.deps.Repo.UpdateLastUsedAt(ctx, ID, time.Now()); err != nil {
		return fmt.Errorf("APITokenUseCase - UpdateLastUsedAt - APITokenRepo.UpdateLastUsedAt: %w", err)
	}
	return nil
}

func (uc *APITokenUseCase) ValidateToken(t *domain.APIToken) bool {
	if t == nil {
		return false
	}
	if t.ExpiresAt != nil && t.ExpiresAt.Before(time.Now()) {
		return false
	}
	return true
}
