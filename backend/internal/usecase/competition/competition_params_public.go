package competition

import (
	"context"
	"fmt"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

// GetPublic returns competition parameters for the public allow-list, falling
// back to registry defaults for keys that have not been persisted yet.
func (uc *CompetitionParamUseCase) GetPublic(ctx context.Context) ([]*domain.CompetitionParam, error) {
	all, err := uc.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("CompetitionParamUseCase - GetPublic - GetAll: %w", err)
	}

	byKey := make(map[string]*domain.CompetitionParam, len(all))
	for _, p := range all {
		byKey[p.Key] = p
	}

	list := make([]*domain.CompetitionParam, 0, len(publicConfigKeys))
	for _, key := range publicConfigKeys {
		if p, ok := byKey[key]; ok {
			list = append(list, p)

			continue
		}

		if def, ok := domain.GetConfigDef(key); ok {
			list = append(list, paramFromDef(def))
		}
	}

	return list, nil
}
