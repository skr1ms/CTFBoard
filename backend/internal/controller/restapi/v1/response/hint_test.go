package response

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func TestFromUnlockListWrapsHintUnlocks(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	hintID := uuid.New()
	teamID := uuid.New()
	challengeID := uuid.New()
	unlockedAt := time.Now().UTC().Truncate(time.Second)

	got := FromUnlockList([]*domain.UnlockWithDetails{{
		ID:          id,
		Type:        domain.UnlockTypeHint,
		ResourceID:  hintID,
		HintID:      hintID,
		TeamID:      teamID,
		UnlockedAt:  unlockedAt,
		ChallengeID: challengeID,
		HintCost:    25,
	}}, 1, 1, 20)

	require.NotNil(t, got.Data)
	require.Len(t, *got.Data, 1)

	item := (*got.Data)[0]
	require.NotNil(t, item.ID)
	require.NotNil(t, item.Type)
	require.NotNil(t, item.ResourceID)
	require.NotNil(t, item.HintID)
	require.NotNil(t, item.TeamID)
	require.NotNil(t, item.UnlockedAt)
	require.NotNil(t, item.ChallengeID)
	require.NotNil(t, item.HintCost)

	assert.Equal(t, id.String(), *item.ID)
	assert.Equal(t, openapi.UnlockResponseTypeHint, *item.Type)
	assert.Equal(t, hintID.String(), *item.ResourceID)
	assert.Equal(t, hintID.String(), *item.HintID)
	assert.Equal(t, teamID.String(), *item.TeamID)
	assert.Equal(t, unlockedAt, *item.UnlockedAt)
	assert.Equal(t, challengeID.String(), *item.ChallengeID)
	assert.Equal(t, 25, *item.HintCost)

	require.NotNil(t, got.Meta)
	require.NotNil(t, got.Meta.Total)
	assert.Equal(t, 1, *got.Meta.Total)
}
