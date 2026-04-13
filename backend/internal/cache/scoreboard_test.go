package cache

import (
	"context"
	"testing"

	"github.com/go-redis/redismock/v9"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/wahrwelt-kit/go-cachekit"

	cacheMock "github.com/TakuyaYagam1/AstroCTFb/internal/cache/mock"
)

func TestNewScoreboardCacheService(t *testing.T) {
	t.Parallel()

	client, _ := redismock.NewClientMock()
	c := cachekit.New(client)
	getter := cacheMock.NewMockTeamBracketIDGetter(t)
	svc := NewScoreboardCacheService(c, getter)
	require.NotNil(t, svc)
}

func TestScoreboardCacheService_InvalidateAll_Success(t *testing.T) {
	t.Parallel()

	client, redisMock := redismock.NewClientMock()
	c := cachekit.New(client)
	svc := NewScoreboardCacheService(c, nil)
	ctx := context.Background()

	redisMock.ExpectDel(KeyScoreboard, KeyScoreboardFrozen).SetVal(0)

	svc.InvalidateAll(ctx)

	require.NoError(t, redisMock.ExpectationsWereMet())
}

func TestScoreboardCacheService_InvalidateAll_NilCache(t *testing.T) {
	t.Parallel()

	svc := NewScoreboardCacheService(nil, nil)
	ctx := context.Background()
	svc.InvalidateAll(ctx)
}

func TestScoreboardCacheService_InvalidateForTeam_Success(t *testing.T) {
	t.Parallel()

	client, redisMock := redismock.NewClientMock()
	c := cachekit.New(client)
	svc := NewScoreboardCacheService(c, nil)
	ctx := context.Background()

	redisMock.ExpectDel(KeyScoreboard, KeyScoreboardFrozen).SetVal(0)

	svc.InvalidateForTeam(ctx, uuid.New())

	require.NoError(t, redisMock.ExpectationsWereMet())
}

func TestScoreboardCacheService_InvalidateForTeam_GetterNil(t *testing.T) {
	t.Parallel()

	client, redisMock := redismock.NewClientMock()
	c := cachekit.New(client)
	svc := NewScoreboardCacheService(c, nil)
	ctx := context.Background()

	redisMock.ExpectDel(KeyScoreboard, KeyScoreboardFrozen).SetVal(0)

	svc.InvalidateForTeam(ctx, uuid.New())

	require.NoError(t, redisMock.ExpectationsWereMet())
}

func TestScoreboardCacheService_InvalidateForTeam_WithGetter(t *testing.T) {
	t.Parallel()

	client, redisMock := redismock.NewClientMock()
	c := cachekit.New(client)
	svc := NewScoreboardCacheService(c, cacheMock.NewMockTeamBracketIDGetter(t))
	ctx := context.Background()

	redisMock.ExpectDel(KeyScoreboard, KeyScoreboardFrozen).SetVal(0)

	svc.InvalidateForTeam(ctx, uuid.New())

	require.NoError(t, redisMock.ExpectationsWereMet())
}
