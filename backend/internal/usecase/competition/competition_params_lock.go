package competition

import (
	"context"
	"fmt"
	"strings"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

// GetForUpdate returns a competition parameter under a database row lock.
// Callers should invoke this inside TransactionManager.Run so the lock is held
// until the surrounding transaction commits or rolls back.
func (uc *CompetitionParamUseCase) GetForUpdate(ctx context.Context, key string) (*domain.CompetitionParam, error) {
	key = strings.TrimSpace(key)
	if err := validateCompetitionParamKey(key); err != nil {
		return nil, err
	}

	p, err := uc.deps.Repo.GetByKeyForUpdate(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("CompetitionParamUseCase - GetForUpdate - Repo.GetByKeyForUpdate: %w", err)
	}

	return p, nil
}
