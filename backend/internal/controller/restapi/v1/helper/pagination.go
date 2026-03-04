package helper

import (
	"context"
	"fmt"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

type SettingsGetter interface {
	Get(ctx context.Context) (*entity.Settings, error)
}

func ResolvePageParams(ctx context.Context, getter SettingsGetter, page, perPage *int) (pageNum, perPageNum int, err error) {
	settings, err := getter.Get(ctx)
	if err != nil {
		return 0, 0, err
	}

	defPP := settings.DefaultPerPage
	if defPP <= 0 {
		defPP = usecase.DefaultPerPage
	}
	maxPP := settings.MaxPerPage
	if maxPP <= 0 {
		maxPP = usecase.DefaultMaxPerPage
	}

	return ClampPage(page), ClampPerPage(perPage, defPP, maxPP), nil
}

func NormalizePerPage(ctx context.Context, getter SettingsGetter, requestedPerPage *int) (int, error) {
	_, perPageNum, err := ResolvePageParams(ctx, getter, nil, requestedPerPage)
	if err != nil {
		return 0, fmt.Errorf("NormalizePerPage - Get: %w", err)
	}
	return perPageNum, nil
}
