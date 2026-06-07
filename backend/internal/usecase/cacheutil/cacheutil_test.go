package cacheutil

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wahrwelt-kit/go-logkit"
)

type cacheutilScoreboardInvalidator struct {
	allCalls      int
	forTeamCalls  int
	liveOnlyCalls int
	teamID        uuid.UUID
}

func (s *cacheutilScoreboardInvalidator) InvalidateAll(context.Context) {
	s.allCalls++
}

func (s *cacheutilScoreboardInvalidator) InvalidateForTeam(_ context.Context, teamID uuid.UUID) {
	s.forTeamCalls++
	s.teamID = teamID
}

func (s *cacheutilScoreboardInvalidator) InvalidateLiveOnly(_ context.Context, teamID uuid.UUID) {
	s.liveOnlyCalls++
	s.teamID = teamID
}

type cacheutilUserInvalidator struct {
	calls  int
	userID uuid.UUID
}

func (u *cacheutilUserInvalidator) InvalidateUser(_ context.Context, userID uuid.UUID) {
	u.calls++
	u.userID = userID
}

type cacheutilChallengeListInvalidator struct {
	allCalls     int
	forTeamCalls int
	teamID       uuid.UUID
}

func (c *cacheutilChallengeListInvalidator) InvalidateAll(context.Context) {
	c.allCalls++
}

func (c *cacheutilChallengeListInvalidator) InvalidateForTeam(_ context.Context, teamID uuid.UUID) {
	c.forTeamCalls++
	c.teamID = teamID
}

type cacheutilContextKey string

func TestKeyBuilders(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "user:u1", KeyUser("u1"))
	assert.Equal(t, "team:t1", KeyTeam("t1"))
	assert.Equal(t, "scoreboard:bracket:b1", KeyScoreboardBracket("b1"))
	assert.Equal(t, "scoreboard:frozen:bracket:b1", KeyScoreboardBracketFrozen("b1"))
	assert.Equal(t, "scoreboard:frozen:123:bracket:b1", KeyScoreboardBracketFrozenAt("b1", 123))
	assert.Equal(t, "scoreboard:frozen:123", KeyScoreboardFrozenAt(123))
	assert.Equal(t, "avatar:user:u1", KeyAvatarUser("u1"))
	assert.Equal(t, "avatar:team:t1", KeyAvatarTeam("t1"))
}

func TestInvalidatorsAreNilSafe(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	assert.NotPanics(t, func() {
		InvalidateUser(ctx, nil, uuid.New())
		InvalidateScoreboard(ctx, nil)
		InvalidateScoreboardForTeam(ctx, nil, uuid.New())
		InvalidateTeam(ctx, nil, nil, uuid.New())
		InvalidateChallengeList(ctx, nil)
		InvalidateWithFreezeAwareness(ctx, nil, uuid.New(), false)
	})
}

func TestInvalidatorsDelegateToPorts(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	userID := uuid.New()
	teamID := uuid.New()

	userCache := &cacheutilUserInvalidator{}
	scoreboardCache := &cacheutilScoreboardInvalidator{}
	challengeListCache := &cacheutilChallengeListInvalidator{}

	InvalidateUser(ctx, userCache, userID)
	InvalidateScoreboard(ctx, scoreboardCache)
	InvalidateScoreboardForTeam(ctx, scoreboardCache, teamID)
	InvalidateChallengeList(ctx, challengeListCache)

	assert.Equal(t, 1, userCache.calls)
	assert.Equal(t, userID, userCache.userID)
	assert.Equal(t, 1, scoreboardCache.allCalls)
	assert.Equal(t, 1, scoreboardCache.forTeamCalls)
	assert.Equal(t, teamID, scoreboardCache.teamID)
	assert.Equal(t, 1, challengeListCache.allCalls)
}

func TestInvalidateWithFreezeAwareness(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	teamID := uuid.New()

	t.Run("live only when frozen", func(t *testing.T) {
		t.Parallel()

		cache := &cacheutilScoreboardInvalidator{}

		InvalidateWithFreezeAwareness(ctx, cache, teamID, true)

		assert.Equal(t, 1, cache.liveOnlyCalls)
		assert.Equal(t, 0, cache.forTeamCalls)
		assert.Equal(t, teamID, cache.teamID)
	})

	t.Run("team cache when not frozen", func(t *testing.T) {
		t.Parallel()

		cache := &cacheutilScoreboardInvalidator{}

		InvalidateWithFreezeAwareness(ctx, cache, teamID, false)

		assert.Equal(t, 0, cache.liveOnlyCalls)
		assert.Equal(t, 1, cache.forTeamCalls)
		assert.Equal(t, teamID, cache.teamID)
	})
}

func TestLoaderContextPreservesValuesAndIgnoresParentCancellation(t *testing.T) {
	t.Parallel()

	parent, cancelParent := context.WithCancel(context.WithValue(context.Background(), cacheutilContextKey("request_id"), "req-1"))
	cancelParent()

	ctx, cancel := LoaderContext(parent)
	defer cancel()

	assert.Equal(t, "req-1", ctx.Value(cacheutilContextKey("request_id")))

	deadline, ok := ctx.Deadline()
	require.True(t, ok)
	assert.WithinDuration(t, time.Now().Add(LoaderTimeout), deadline, time.Second)

	select {
	case <-ctx.Done():
		t.Fatal("loader context should not inherit parent cancellation")
	default:
	}
}

type cacheutilPubSubFake struct {
	ch chan string
}

func (p *cacheutilPubSubFake) Publish(context.Context, string, string) error {
	return nil
}

func (p *cacheutilPubSubFake) Subscribe(context.Context, string) (<-chan string, error) {
	return p.ch, nil
}

func TestSubscribeInvalidationHandlesMessageAndStops(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pubsub := &cacheutilPubSubFake{ch: make(chan string, 1)}
	pubsub.ch <- "invalidate"

	calls := 0

	SubscribeInvalidation(ctx, pubsub, PubSubScoreboard, func() {
		calls++

		cancel()
	}, logkit.Noop(), "test")

	assert.Equal(t, 1, calls)
}
