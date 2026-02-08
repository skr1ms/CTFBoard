package cache

import (
	"context"
	"testing"

	"github.com/go-redis/redismock/v9"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewScoreboardCacheService(t *testing.T) {
	client, _ := redismock.NewClientMock()
	c := New(client)
	getter := &mockBracketGetter{bracketID: nil, err: nil}
	svc := NewScoreboardCacheService(c, getter)
	require.NotNil(t, svc)
}

func TestScoreboardCacheService_InvalidateAll_Success(t *testing.T) {
	client, mock := redismock.NewClientMock()
	c := New(client)
	svc := NewScoreboardCacheService(c, nil)
	ctx := context.Background()

	mock.ExpectDel(KeyScoreboard, KeyScoreboardFrozen).SetVal(0)

	svc.InvalidateAll(ctx)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestScoreboardCacheService_InvalidateAll_NilCache(t *testing.T) {
	svc := NewScoreboardCacheService(nil, nil)
	ctx := context.Background()
	svc.InvalidateAll(ctx)
}

func TestScoreboardCacheService_InvalidateForTeam_Success(t *testing.T) {
	client, mock := redismock.NewClientMock()
	c := New(client)
	bracketID := uuid.New()
	getter := &mockBracketGetter{bracketID: &bracketID, err: nil}
	svc := NewScoreboardCacheService(c, getter)
	ctx := context.Background()
	teamID := uuid.New()

	mock.ExpectDel(KeyScoreboard, KeyScoreboardFrozen).SetVal(0)
	mock.ExpectDel(KeyScoreboardBracket(bracketID.String()), KeyScoreboardBracketFrozen(bracketID.String())).SetVal(0)

	svc.InvalidateForTeam(ctx, teamID)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestScoreboardCacheService_InvalidateForTeam_GetterNil(t *testing.T) {
	client, mock := redismock.NewClientMock()
	c := New(client)
	svc := NewScoreboardCacheService(c, nil)
	ctx := context.Background()

	mock.ExpectDel(KeyScoreboard, KeyScoreboardFrozen).SetVal(0)

	svc.InvalidateForTeam(ctx, uuid.New())

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestScoreboardCacheService_InvalidateForTeam_GetterError(t *testing.T) {
	client, mock := redismock.NewClientMock()
	c := New(client)
	getter := &mockBracketGetter{bracketID: nil, err: assert.AnError}
	svc := NewScoreboardCacheService(c, getter)
	ctx := context.Background()

	mock.ExpectDel(KeyScoreboard, KeyScoreboardFrozen).SetVal(0)

	svc.InvalidateForTeam(ctx, uuid.New())

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestScoreboardCacheService_InvalidateForTeam_GetterReturnsNilBracket(t *testing.T) {
	client, mock := redismock.NewClientMock()
	c := New(client)
	getter := &mockBracketGetter{bracketID: nil, err: nil}
	svc := NewScoreboardCacheService(c, getter)
	ctx := context.Background()

	mock.ExpectDel(KeyScoreboard, KeyScoreboardFrozen).SetVal(0)

	svc.InvalidateForTeam(ctx, uuid.New())

	require.NoError(t, mock.ExpectationsWereMet())
}

type mockBracketGetter struct {
	bracketID *uuid.UUID
	err       error
}

func (m *mockBracketGetter) GetTeamBracketID(ctx context.Context, teamID uuid.UUID) (*uuid.UUID, error) {
	return m.bracketID, m.err
}
