package team

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

func TestTeamUseCase_GetInviteToken_Success(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	captainID := uuid.New()
	teamID := uuid.New()
	user := newTestUser(captainID, &teamID, "cap", "cap@x.com")
	team := newTestTeam(teamID, "MyTeam", captainID, uuid.New(), false)

	d.compRepo.EXPECT().Get(mock.Anything).Return(&domain.Competition{Mode: "teams_only", AllowTeamSwitch: true}, nil).Once()
	d.userRepo.EXPECT().GetByID(mock.Anything, captainID).Return(user, nil).Once()
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(team, nil).Once()

	uc := d.createUseCase()

	result, err := uc.GetInviteToken(context.Background(), captainID)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, teamID, result.ID)
	assert.Equal(t, captainID, result.CaptainID)
}

func TestTeamUseCase_GetInviteToken_Error_NotCaptain(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	captainID := uuid.New()
	otherCaptainID := uuid.New()
	teamID := uuid.New()
	user := newTestUser(captainID, &teamID, "member", "m@x.com")
	team := newTestTeam(teamID, "MyTeam", otherCaptainID, uuid.New(), false)

	d.compRepo.EXPECT().Get(mock.Anything).Return(&domain.Competition{Mode: "teams_only", AllowTeamSwitch: true}, nil).Once()
	d.userRepo.EXPECT().GetByID(mock.Anything, captainID).Return(user, nil).Once()
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(team, nil).Once()

	uc := d.createUseCase()

	result, err := uc.GetInviteToken(context.Background(), captainID)

	assert.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrNotCaptain)
	assert.Nil(t, result)
}

func TestTeamUseCase_UpdateMyTeam_Success(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	captainID := uuid.New()
	teamID := uuid.New()
	user := newTestUser(captainID, &teamID, "cap", "cap@x.com")
	team := newTestTeam(teamID, "OldName", captainID, uuid.New(), false)

	d.compRepo.EXPECT().Get(mock.Anything).Return(&domain.Competition{Mode: "teams_only", AllowTeamSwitch: true}, nil).Times(2)
	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.userRepo.EXPECT().Lock(mock.Anything, captainID).Return(nil).Once()
	d.userRepo.EXPECT().GetByID(mock.Anything, captainID).Return(user, nil).Once()
	d.teamRepo.EXPECT().Lock(mock.Anything, teamID).Return(nil).Once()
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(team, nil).Once()
	d.teamRepo.EXPECT().GetByName(mock.Anything, "NewName").Return(nil, apperr.ErrTeamNotFound).Once()
	d.teamRepo.EXPECT().UpdateName(mock.Anything, teamID, "NewName").Return(nil).Once()

	uc := d.createUseCase()

	result, err := uc.UpdateMyTeam(context.Background(), captainID, "NewName")

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "NewName", result.Name)
}

func TestTeamUseCase_UpdateMyTeam_Error_NotCaptain(t *testing.T) {
	t.Parallel()
	d := newTeamTestDeps(t)

	captainID := uuid.New()
	realCaptainID := uuid.New()
	teamID := uuid.New()
	user := newTestUser(captainID, &teamID, "member", "m@x.com")
	team := newTestTeam(teamID, "Team", realCaptainID, uuid.New(), false)

	d.compRepo.EXPECT().Get(mock.Anything).Return(&domain.Competition{Mode: "teams_only", AllowTeamSwitch: true}, nil).Times(2)
	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.userRepo.EXPECT().Lock(mock.Anything, captainID).Return(nil).Once()
	d.userRepo.EXPECT().GetByID(mock.Anything, captainID).Return(user, nil).Once()
	d.teamRepo.EXPECT().Lock(mock.Anything, teamID).Return(nil).Once()
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(team, nil).Once()

	uc := d.createUseCase()

	result, err := uc.UpdateMyTeam(context.Background(), captainID, "NewName")

	assert.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrNotCaptain)
	assert.Nil(t, result)
}
