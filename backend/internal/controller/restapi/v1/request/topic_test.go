package request

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/validator"
)

func TestCreateTopicRequest_Validation(t *testing.T) {
	t.Parallel()

	v, err := validator.New()
	require.NoError(t, err)

	assert.NoError(t, v.Validate(&openapi.CreateTopicRequest{Name: "Web"}))
	assert.Error(t, v.Validate(&openapi.CreateTopicRequest{Name: ""}))
	assert.Error(t, v.Validate(&openapi.CreateTopicRequest{Name: string(make([]byte, 101))}))
}

func TestSetChallengeTopicsRequestToParams(t *testing.T) {
	t.Parallel()

	topicID := uuid.New().String()
	got, err := SetChallengeTopicsRequestToParams(&openapi.SetChallengeTopicsRequest{TopicIds: &[]string{topicID}})

	require.NoError(t, err)
	assert.Equal(t, []string{topicID}, got)
}

func TestParseTopicIDs_InvalidTopicID(t *testing.T) {
	t.Parallel()

	raw := []string{"bad-topic-id"}
	_, err := ParseUUIDSlice(&raw, "topic_id")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "topic_id")
}
