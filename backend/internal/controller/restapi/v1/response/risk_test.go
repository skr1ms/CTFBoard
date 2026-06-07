package response

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

func TestFromStorageListKeepsIndependentPathPointers(t *testing.T) {
	t.Parallel()

	got := FromStorageList([]string{"a.txt", "dir/b.txt"})

	require.NotNil(t, got.Objects)
	require.NotNil(t, got.Total)
	require.Len(t, *got.Objects, 2)
	assert.Equal(t, 2, *got.Total)
	assert.Equal(t, "a.txt", *(*got.Objects)[0].Path)
	assert.Equal(t, "dir/b.txt", *(*got.Objects)[1].Path)
	assert.NotSame(t, (*got.Objects)[0].Path, (*got.Objects)[1].Path)
}

func TestFromUserForMePasswordAndBanSemantics(t *testing.T) {
	t.Parallel()

	teamID := uuid.New()
	bannedAt := time.Unix(1000, 0)
	reason := "abuse"
	avatarURL := "https://cdn.example/avatar.png"
	user := &domain.User{
		ID:           uuid.New(),
		TeamID:       &teamID,
		Username:     "alice",
		Email:        "alice@example.com",
		PasswordHash: domain.OAuthOnlyPasswordSentinel,
		Role:         domain.RoleUser,
		AvatarURL:    &avatarURL,
		IsBanned:     true,
		BannedAt:     &bannedAt,
		BannedReason: &reason,
	}

	got := FromUserForMe(user)

	require.NotNil(t, got.ID)
	require.NotNil(t, got.TeamID)
	require.NotNil(t, got.HasPassword)
	require.NotNil(t, got.BanStatus)
	assert.Equal(t, user.ID.String(), *got.ID)
	assert.Equal(t, teamID.String(), *got.TeamID)
	assert.False(t, *got.HasPassword)
	assert.Equal(t, avatarURL, *got.AvatarURL)
	assert.True(t, *got.BanStatus.IsBanned)
	assert.Equal(t, reason, *got.BanStatus.Reason)
	require.NotNil(t, got.BanStatus.BannedAt)
	assert.Equal(t, bannedAt.Format(time.RFC3339), *got.BanStatus.BannedAt)

	user.PasswordHash = "bcrypt-hash"
	got = FromUserForMe(user)
	require.NotNil(t, got.HasPassword)
	assert.True(t, *got.HasPassword)
}

func TestFromTokenPairNilAndValues(t *testing.T) {
	t.Parallel()

	assert.Nil(t, FromTokenPair(nil).AccessToken)
	assert.Nil(t, FromTokenPair(nil).AccessExpiresAt)

	got := FromTokenPair(&usecase.TokenPair{AccessToken: "access", RefreshToken: "refresh", AccessExpiresAt: 123, RefreshExpiresAt: 456})

	require.NotNil(t, got.AccessToken)
	require.NotNil(t, got.AccessExpiresAt)
	assert.Equal(t, "access", *got.AccessToken)
	assert.Equal(t, 123, *got.AccessExpiresAt)
}
