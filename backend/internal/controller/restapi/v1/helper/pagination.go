package helper

import (
	"context"

	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

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

	return httputil.ClampPage(page), httputil.ClampPerPage(perPage, defPP, maxPP), nil
}
