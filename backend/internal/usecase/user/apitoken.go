package user

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/singleflight"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/crypto"
)

const (
	apiTokenDescriptionMaxLen = 255
	apiTokenRandomBytes       = 32
	apiTokenLastUsedMinTTL    = time.Minute
)

type APITokenUseCase struct {
	deps APITokenDeps

	lastUsedMu sync.Mutex
	lastUsedAt map[uuid.UUID]time.Time
	lastUsedSF singleflight.Group
}

type APITokenDeps struct {
	Repo                      repo.APITokenRepository
	Now                       func() time.Time
	LastUsedUpdateMinInterval time.Duration
}

var _ usecase.APITokenUseCase = (*APITokenUseCase)(nil)

func NewAPITokenUseCase(deps APITokenDeps) *APITokenUseCase {
	if deps.Now == nil {
		deps.Now = time.Now
	}

	if deps.LastUsedUpdateMinInterval <= 0 {
		deps.LastUsedUpdateMinInterval = apiTokenLastUsedMinTTL
	}

	return &APITokenUseCase{deps: deps, lastUsedAt: make(map[uuid.UUID]time.Time)}
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
		return "", nil, apperr.NewValidationErrorf("description must not exceed %d characters", apiTokenDescriptionMaxLen)
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
		CreatedAt:   uc.now(),
	}
	if err := uc.deps.Repo.Create(ctx, token); err != nil {
		return "", nil, fmt.Errorf("APITokenUseCase - Create - APITokenRepo.Create: %w", err)
	}

	return plaintext, token, nil
}

func (uc *APITokenUseCase) Delete(ctx context.Context, ID, userID uuid.UUID) error {
	err := uc.deps.Repo.Delete(ctx, ID, userID)
	if err != nil {
		return fmt.Errorf("APITokenUseCase - Delete - APITokenRepo.Delete: %w", err)
	}

	return nil
}

func (uc *APITokenUseCase) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	if err := uc.deps.Repo.DeleteAllByUserID(ctx, userID); err != nil {
		return fmt.Errorf("APITokenUseCase - RevokeAllForUser - APITokenRepo.DeleteAllByUserID: %w", err)
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

func (uc *APITokenUseCase) AuthenticatePlaintext(ctx context.Context, plaintext string) (*domain.APIToken, error) {
	plaintext = strings.TrimSpace(plaintext)
	if plaintext == "" {
		return nil, apperr.ErrInvalidToken
	}

	tokenHash := crypto.SHA256Hex(plaintext)

	token, err := uc.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		return nil, err
	}

	if !uc.ValidateToken(token) {
		return nil, apperr.ErrInvalidToken
	}

	return token, nil
}

func (uc *APITokenUseCase) UpdateLastUsedAt(ctx context.Context, ID uuid.UUID) error {
	if !uc.shouldUpdateLastUsedAt(ID, uc.now()) {
		return nil
	}

	_, err, _ := uc.lastUsedSF.Do(ID.String(), func() (any, error) {
		now := uc.now()
		if !uc.shouldUpdateLastUsedAt(ID, now) {
			return nil, nil
		}

		if err := uc.deps.Repo.UpdateLastUsedAt(ctx, ID, now); err != nil {
			return nil, fmt.Errorf("APITokenUseCase - UpdateLastUsedAt - APITokenRepo.UpdateLastUsedAt: %w", err)
		}

		uc.markLastUsedAt(ID, now)

		return nil, nil
	})

	return err
}

func (uc *APITokenUseCase) ValidateToken(t *domain.APIToken) bool {
	if t == nil {
		return false
	}

	if t.ExpiresAt != nil && t.ExpiresAt.Before(uc.now()) {
		return false
	}

	return true
}

func (uc *APITokenUseCase) now() time.Time {
	return uc.deps.Now()
}

func (uc *APITokenUseCase) shouldUpdateLastUsedAt(ID uuid.UUID, now time.Time) bool {
	uc.lastUsedMu.Lock()
	defer uc.lastUsedMu.Unlock()

	last, ok := uc.lastUsedAt[ID]

	return !ok || now.Sub(last) >= uc.deps.LastUsedUpdateMinInterval
}

func (uc *APITokenUseCase) markLastUsedAt(ID uuid.UUID, at time.Time) {
	uc.lastUsedMu.Lock()
	defer uc.lastUsedMu.Unlock()

	uc.lastUsedAt[ID] = at
}
