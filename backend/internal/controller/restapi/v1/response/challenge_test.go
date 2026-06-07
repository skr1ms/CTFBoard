package response

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

func TestFromChallenge_IncludesMetadata(t *testing.T) {
	t.Parallel()

	nextID := uuid.New()
	challenge := &domain.Challenge{
		ID:              uuid.New(),
		Title:           "challenge",
		Description:     "desc",
		Category:        "web",
		Attribution:     "author",
		ConnectionInfo:  "nc host 31337",
		NextChallengeID: &nextID,
		State:           domain.ChallengeStateVisible,
	}

	got := FromChallenge(challenge)

	require.NotNil(t, got.Attribution)
	assert.Equal(t, "author", *got.Attribution)
	require.NotNil(t, got.NextID)
	assert.Equal(t, nextID, uuid.UUID(*got.NextID))
}

func TestFromChallengeWithTags_UnmetRequirementsHidesMetadata(t *testing.T) {
	t.Parallel()

	nextID := uuid.New()
	requirementsMet := false
	challenge := &domain.Challenge{
		ID:              uuid.New(),
		Title:           "challenge",
		Description:     "desc",
		Category:        "web",
		Attribution:     "author",
		ConnectionInfo:  "nc host 31337",
		NextChallengeID: &nextID,
		State:           domain.ChallengeStateVisible,
	}

	got := FromChallengeWithTags(&usecase.ChallengeWithTags{
		ChallengeWithSolved: &domain.ChallengeWithSolved{Challenge: challenge},
		RequirementsMet:     &requirementsMet,
	})

	assert.Nil(t, got.Attribution)
	assert.Nil(t, got.ConnectionInfo)
	assert.Nil(t, got.NextID)
}
