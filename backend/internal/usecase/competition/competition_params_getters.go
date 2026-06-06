package competition

import (
	"context"
	"errors"
	"strconv"

	"github.com/wahrwelt-kit/go-logkit"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
)

func (uc *CompetitionParamUseCase) GetString(ctx context.Context, key, defaultVal string) string {
	p, err := uc.Get(ctx, key)
	if err != nil {
		if errors.Is(err, apperr.ErrCompetitionParamNotFound) {
			return defaultVal
		}

		uc.deps.Logger.WithError(err).Warn("competition_params: GetString failed, returning default", logkit.Fields{"key": key})

		return defaultVal
	}

	if p == nil {
		return defaultVal
	}

	return p.Value
}

func (uc *CompetitionParamUseCase) GetInt(ctx context.Context, key string, defaultVal int) int {
	p, err := uc.Get(ctx, key)
	if err != nil {
		if errors.Is(err, apperr.ErrCompetitionParamNotFound) {
			return defaultVal
		}

		uc.deps.Logger.WithError(err).Warn("competition_params: GetInt failed, returning default", logkit.Fields{"key": key})

		return defaultVal
	}

	if p == nil {
		return defaultVal
	}

	val, err := strconv.Atoi(p.Value)
	if err != nil {
		return defaultVal
	}

	return val
}

func (uc *CompetitionParamUseCase) GetBool(ctx context.Context, key string, defaultVal bool) bool {
	p, err := uc.Get(ctx, key)
	if err != nil {
		if errors.Is(err, apperr.ErrCompetitionParamNotFound) {
			return defaultVal
		}

		uc.deps.Logger.WithError(err).Warn("competition_params: GetBool failed, returning default", logkit.Fields{"key": key})

		return defaultVal
	}

	if p == nil {
		return defaultVal
	}

	return p.Value == "true"
}
