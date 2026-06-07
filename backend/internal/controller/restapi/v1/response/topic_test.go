package response

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

func TestFromTopic(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	createdAt := time.Now().UTC()
	got := FromTopic(&domain.Topic{ID: id, Name: "web", CreatedAt: createdAt})

	require.NotNil(t, got.ID)
	require.NotNil(t, got.Name)
	require.NotNil(t, got.CreatedAt)
	assert.Equal(t, id.String(), *got.ID)
	assert.Equal(t, "web", *got.Name)
	assert.Equal(t, createdAt, *got.CreatedAt)
}
