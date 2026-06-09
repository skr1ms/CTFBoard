package response

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
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
	require.NotNil(t, got.BanStatus.Source)
	assert.Equal(t, "direct", string(*got.BanStatus.Source))
	assert.Equal(t, reason, *got.BanStatus.Reason)
	require.NotNil(t, got.BanStatus.BannedAt)
	assert.Equal(t, bannedAt.Format(time.RFC3339), *got.BanStatus.BannedAt)

	user.PasswordHash = "bcrypt-hash"
	got = FromUserForMe(user)
	require.NotNil(t, got.HasPassword)
	assert.True(t, *got.HasPassword)
}

func TestFromUserForMeInheritedTeamBanSemantics(t *testing.T) {
	t.Parallel()

	user := &domain.User{
		ID:              uuid.New(),
		Username:        "alice",
		Email:           "alice@example.com",
		Role:            domain.RoleUser,
		WasInBannedTeam: true,
	}

	got := FromUserForMe(user)

	require.NotNil(t, got.BanStatus)
	require.NotNil(t, got.BanStatus.IsBanned)
	require.NotNil(t, got.BanStatus.Source)
	require.NotNil(t, got.BanStatus.CanAppeal)
	require.NotNil(t, got.BanStatus.HasPendingAppeal)
	assert.True(t, *got.BanStatus.IsBanned)
	assert.Equal(t, "team_inherited", string(*got.BanStatus.Source))
	assert.False(t, *got.BanStatus.CanAppeal)
	assert.False(t, *got.BanStatus.HasPendingAppeal)
	assert.Nil(t, got.BanStatus.Reason)
	assert.Nil(t, got.BanStatus.BannedAt)
}

func TestUserResponsesExposeCustomFields(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	fieldID := uuid.New().String()
	customFields := usecase.CustomFieldValues{fieldID: map[string]any{"role": "captain"}}
	user := &domain.User{ID: userID, Username: "alice", Email: "alice@example.com"}

	me := FromUserMe(&usecase.UserMe{User: user, CustomFields: customFields})
	require.NotNil(t, me.CustomFields)
	assert.Equal(t, customFields, *me.CustomFields)

	profile := FromUserProfile(&usecase.UserProfile{User: user, CustomFields: customFields})
	require.NotNil(t, profile.CustomFields)
	assert.Equal(t, customFields, *profile.CustomFields)
}

func TestFromAdminUserBanSourceSemantics(t *testing.T) {
	t.Parallel()

	direct := FromAdminUser(&domain.User{
		ID:       uuid.New(),
		Username: "alice",
		Email:    "alice@example.com",
		Role:     domain.RoleUser,
		IsBanned: true,
	})
	require.NotNil(t, direct.IsBanned)
	require.NotNil(t, direct.WasInBannedTeam)
	require.NotNil(t, direct.IsBlocked)
	require.NotNil(t, direct.BanSource)
	assert.True(t, *direct.IsBanned)
	assert.False(t, *direct.WasInBannedTeam)
	assert.True(t, *direct.IsBlocked)
	assert.Equal(t, openapi.AdminUserResponseBanSourceDirect, *direct.BanSource)

	inherited := FromAdminUser(&domain.User{
		ID:              uuid.New(),
		Username:        "bob",
		Email:           "bob@example.com",
		Role:            domain.RoleUser,
		WasInBannedTeam: true,
	})
	require.NotNil(t, inherited.IsBlocked)
	require.NotNil(t, inherited.BanSource)
	assert.True(t, *inherited.IsBlocked)
	assert.Equal(t, openapi.AdminUserResponseBanSourceTeamInherited, *inherited.BanSource)

	adminFromBannedTeam := FromAdminUser(&domain.User{
		ID:              uuid.New(),
		Username:        "root",
		Email:           "root@example.com",
		Role:            domain.RoleAdmin,
		WasInBannedTeam: true,
	})
	require.NotNil(t, adminFromBannedTeam.IsBlocked)
	require.NotNil(t, adminFromBannedTeam.BanSource)
	assert.False(t, *adminFromBannedTeam.IsBlocked)
	assert.Equal(t, openapi.AdminUserResponseBanSourceNone, *adminFromBannedTeam.BanSource)
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
