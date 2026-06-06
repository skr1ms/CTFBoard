package avatar

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/cacheutil"
)

// resolvePresignedURLs checks the Redis cache for pre-signed avatar URLs and,
// on a miss, fetches them from storage and writes the result back to cache.
// avatarURL must be non-nil and non-empty (callers are responsible for the guard).
func (uc *AvatarUseCase) resolvePresignedURLs(ctx context.Context, cacheKey string, avatarURL *string) (fullURL, thumbURL *string, err error) {
	if fullURL, thumbURL, ok := uc.cachedPresignedURLs(ctx, cacheKey); ok {
		return fullURL, thumbURL, nil
	}

	fullPath := *avatarURL
	thumbPath := domain.ThumbPathFromFull(fullPath)

	result, err, _ := uc.presignSF.Do(cacheKey, func() (any, error) {
		loadCtx, cancel := cacheutil.LoaderContext(ctx)
		defer cancel()

		if fullURL, thumbURL, ok := uc.cachedPresignedURLs(loadCtx, cacheKey); ok {
			return cachedAvatarURLs{FullURL: *fullURL, ThumbURL: *thumbURL}, nil
		}

		fullPresigned, err := uc.deps.Storage.GetPresignedURL(loadCtx, fullPath, uc.deps.Config.PresignedTTL)
		if err != nil {
			return nil, fmt.Errorf("AvatarUseCase - resolvePresignedURLs - GetPresignedURL full: %w", err)
		}

		thumbPresigned, err := uc.deps.Storage.GetPresignedURL(loadCtx, thumbPath, uc.deps.Config.PresignedTTL)
		if err != nil {
			return nil, fmt.Errorf("AvatarUseCase - resolvePresignedURLs - GetPresignedURL thumb: %w", err)
		}

		cached := cachedAvatarURLs{FullURL: fullPresigned, ThumbURL: thumbPresigned}
		if cacheBytes, _ := json.Marshal(cached); cacheBytes != nil {
			_ = uc.deps.Cache.Set(loadCtx, cacheKey, cacheBytes, uc.deps.Config.CacheTTL)
		}

		return cached, nil
	})
	if err != nil {
		return nil, nil, err
	}

	urls, ok := result.(cachedAvatarURLs)
	if !ok {
		return nil, nil, fmt.Errorf("AvatarUseCase - resolvePresignedURLs - unexpected result type")
	}

	return &urls.FullURL, &urls.ThumbURL, nil
}

func (uc *AvatarUseCase) cachedPresignedURLs(ctx context.Context, cacheKey string) (*string, *string, bool) {
	rawCache, err := uc.deps.Cache.Get(ctx, cacheKey)
	if err != nil {
		return nil, nil, false
	}

	var cached cachedAvatarURLs
	if err := json.Unmarshal(rawCache, &cached); err != nil {
		return nil, nil, false
	}

	if cached.FullURL == "" || cached.ThumbURL == "" {
		return nil, nil, false
	}

	return &cached.FullURL, &cached.ThumbURL, true
}
