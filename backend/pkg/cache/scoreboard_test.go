package cache

import (
	"context"
	"testing"

	cachemocks "github.com/TakuyaYagam1/AstroCTFb/pkg/cache/mocks"
	"github.com/go-redis/redismock/v9"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNewScoreboardCacheService(t *testing.T) {
	t.Parallel()
	client, _ := redismock.NewClientMock()
	c := New(client)
	getter := cachemocks.NewMockTeamBracketIDGetter(t)
	svc := NewScoreboardCacheService(c, getter)
	require.NotNil(t, svc)
}

func TestScoreboardCacheService_InvalidateAll_Success(t *testing.T) {
	t.Parallel()
	client, redisMock := redismock.NewClientMock()
	c := New(client)
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
	c := New(client)
	bracketID := uuid.New()
	teamID := uuid.New()

	getter := cachemocks.NewMockTeamBracketIDGetter(t)
	getter.On("GetTeamBracketID", mock.Anything, teamID).Return(&bracketID, nil)

	svc := NewScoreboardCacheService(c, getter)
	ctx := context.Background()

	redisMock.ExpectDel(KeyScoreboard, KeyScoreboardFrozen).SetVal(0)
	redisMock.ExpectDel(KeyScoreboardBracket(bracketID.String()), KeyScoreboardBracketFrozen(bracketID.String())).SetVal(0)

	svc.InvalidateForTeam(ctx, teamID)

	require.NoError(t, redisMock.ExpectationsWereMet())
}

func TestScoreboardCacheService_InvalidateForTeam_GetterNil(t *testing.T) {
	t.Parallel()
	client, redisMock := redismock.NewClientMock()
	c := New(client)
	svc := NewScoreboardCacheService(c, nil)
	ctx := context.Background()

	redisMock.ExpectDel(KeyScoreboard, KeyScoreboardFrozen).SetVal(0)

	svc.InvalidateForTeam(ctx, uuid.New())

	require.NoError(t, redisMock.ExpectationsWereMet())
}

func TestScoreboardCacheService_InvalidateForTeam_GetterError(t *testing.T) {
	t.Parallel()
	client, redisMock := redismock.NewClientMock()
	c := New(client)
	teamID := uuid.New()

	getter := cachemocks.NewMockTeamBracketIDGetter(t)
	getter.On("GetTeamBracketID", mock.Anything, teamID).Return((*uuid.UUID)(nil), assert.AnError)

	svc := NewScoreboardCacheService(c, getter)
	ctx := context.Background()

	redisMock.ExpectDel(KeyScoreboard, KeyScoreboardFrozen).SetVal(0)

	svc.InvalidateForTeam(ctx, teamID)

	require.NoError(t, redisMock.ExpectationsWereMet())
}

func TestScoreboardCacheService_InvalidateForTeam_GetterReturnsNilBracket(t *testing.T) {
	t.Parallel()
	client, redisMock := redismock.NewClientMock()
	c := New(client)
	teamID := uuid.New()

	getter := cachemocks.NewMockTeamBracketIDGetter(t)
	getter.On("GetTeamBracketID", mock.Anything, teamID).Return((*uuid.UUID)(nil), nil)

	svc := NewScoreboardCacheService(c, getter)
	ctx := context.Background()

	redisMock.ExpectDel(KeyScoreboard, KeyScoreboardFrozen).SetVal(0)

	svc.InvalidateForTeam(ctx, teamID)

	require.NoError(t, redisMock.ExpectationsWereMet())
}
