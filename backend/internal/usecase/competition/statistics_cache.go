package competition

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/wahrwelt-kit/go-cachekit"

	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/cacheutil"
)

func statsCachedLoad[T any](
	uc *StatisticsUseCase,
	ctx context.Context,
	key string,
	ttl time.Duration,
	fn func(context.Context) (T, error),
) (T, error) {
	var zero T

	v, err, _ := uc.sf.Do(key, func() (any, error) {
		loadCtx, cancel := cacheutil.LoaderContext(ctx)
		defer cancel()

		return cachekit.GetOrLoad(uc.deps.Cache, loadCtx, key, ttl, fn)
	})
	if err != nil {
		return zero, err
	}

	result, ok := v.(T)
	if !ok {
		return zero, fmt.Errorf("StatisticsUseCase - statsCachedLoad: unexpected cached type for key %s", key)
	}

	return result, nil
}

func (uc *StatisticsUseCase) readOnly(ctx context.Context, op string, fn func(context.Context) error) error {
	if uc.deps.TM == nil {
		return fn(ctx)
	}

	if err := uc.deps.TM.ReadOnly(ctx, fn); err != nil {
		return fmt.Errorf("StatisticsUseCase - %s - TM.ReadOnly: %w", op, err)
	}

	return nil
}

func statsReadOnlyLoad[T any](
	uc *StatisticsUseCase,
	ctx context.Context,
	op string,
	repoOp string,
	fn func(context.Context) (T, error),
) (T, error) {
	var result T

	err := uc.readOnly(ctx, op, func(roCtx context.Context) error {
		var err error

		result, err = fn(roCtx)
		if err != nil {
			return fmt.Errorf("StatisticsUseCase - %s - %s: %w", op, repoOp, err)
		}

		return nil
	})
	if err != nil {
		var zero T

		return zero, err
	}

	return result, nil
}

// isFrozen checks whether the competition scoreboard freeze is currently active.
// Returns (false, zero) when CompGetter is nil, when the competition cannot be
// loaded, or when IsFreezeActive returns false.
func (uc *StatisticsUseCase) isFrozen(ctx context.Context) (bool, time.Time) {
	if uc.deps.CompGetter == nil {
		return false, time.Time{}
	}

	comp, err := uc.deps.CompGetter.Get(ctx)
	if err != nil || comp == nil {
		return false, time.Time{}
	}

	if comp.IsFreezeActive() {
		return true, *comp.FreezeTime
	}

	return false, time.Time{}
}

func statsFrozenSuffix(freezeTime time.Time) string {
	return ":frozen:" + strconv.FormatInt(freezeTime.Unix(), 10)
}

// freezeAwareLoad is a generic helper that handles the repeated pattern across
// StatisticsUseCase methods: resolve freeze state, build a freeze-encoded cache
// key, and call cachekit.GetOrLoad through singleflight using a loader context
// that is decoupled from the first caller's cancellation.
func freezeAwareLoad[T any](
	uc *StatisticsUseCase,
	ctx context.Context,
	baseKey string,
	ttl time.Duration,
	frozen bool,
	freezeTime time.Time,
	fn func(context.Context, *time.Time) (T, error),
) (T, error) {
	key := baseKey

	var ft *time.Time

	if frozen {
		key = baseKey + statsFrozenSuffix(freezeTime)
		ft = &freezeTime
	}

	return statsCachedLoad(uc, ctx, key, ttl, func(ctx context.Context) (T, error) {
		return fn(ctx, ft)
	})
}
